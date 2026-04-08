import argparse
import base64
import json
import logging
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Literal, Optional

import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Volcengine 豆包 openspeech HTTP v3 单向流（与 cmd/volc-tts、pkg/langgraphcompat 一致）
VOLC_V3_DEFAULT_ENDPOINT = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
VOLC_V3_ADDITIONS = (
    '{"disable_markdown_filter":true,"enable_language_detector":true,"enable_latex_tn":true,'
    '"disable_default_bit_rate":true,"max_length_to_filter_parenthesis":0,'
    '"cache_config":{"text_type":1,"use_cache":true}}'
)


# Types
class ScriptLine:
    def __init__(self, speaker: Literal["male", "female"] = "male", paragraph: str = ""):
        self.speaker = speaker
        self.paragraph = paragraph


class Script:
    def __init__(self, locale: Literal["en", "zh"] = "en", lines: Optional[list[ScriptLine]] = None):
        self.locale = locale
        self.lines = lines or []

    @classmethod
    def from_dict(cls, data: dict) -> "Script":
        script = cls(locale=data.get("locale", "en"))
        for line in data.get("lines", []):
            script.lines.append(
                ScriptLine(
                    speaker=line.get("speaker", "male"),
                    paragraph=line.get("paragraph", ""),
                )
            )
        return script


def _resolve_volc_api_key() -> str:
    """Match gatewayVolcTTSConfigFromEnv: API key for x-api-key header."""
    key = (
        os.getenv("VOLCENGINE_TTS_API_KEY", "").strip()
        or os.getenv("TTS_API_KEY", "").strip()
    )
    if key:
        return key
    token = os.getenv("VOLCENGINE_TTS_ACCESS_TOKEN", "").strip()
    if token:
        return token
    app_id = os.getenv("VOLCENGINE_TTS_APP_ID", "").strip()
    if app_id.lower().startswith("api-key-"):
        return app_id
    return ""


def _resource_id_derived_from_voice(voice: str) -> str:
    """Match volcV3ResourceIDByVoice in pkg/langgraphcompat/tts_volcengine_v3_unidirectional.go."""
    v = voice.strip()
    low = v.lower()
    if v.startswith("S_"):
        return "volc.megatts.default"
    if low.startswith("seed-tts"):
        return v
    if "seed_tts" in low:
        return v
    return "volc.service_type.10029"


def _resource_id_for_request(voice: str) -> str:
    explicit = os.getenv("VOLCENGINE_TTS_RESOURCE_ID", "").strip()
    if explicit:
        return explicit
    return _resource_id_derived_from_voice(voice)


def _normalize_audio_format(fmt: str) -> str:
    f = fmt.strip().lower()
    return f if f in ("wav", "pcm") else "mp3"


def _parse_volc_v3_stream_body(body: str) -> Optional[bytes]:
    """Decode newline or concatenated JSON stream; aggregate base64 audio chunks."""
    dec = json.JSONDecoder()
    idx = 0
    audio = bytearray()
    last_code = 0
    last_msg = ""
    n = len(body)
    while idx < n:
        while idx < n and body[idx].isspace():
            idx += 1
        if idx >= n:
            break
        try:
            obj, end = dec.raw_decode(body, idx)
        except json.JSONDecodeError as e:
            logger.error(f"TTS stream JSON decode at {idx}: {e}")
            return None
        idx = end
        last_code = int(obj.get("code", 0))
        last_msg = str(obj.get("message") or "").strip()
        if last_code not in (0, 20_000_000):
            logger.error(f"TTS error: code={last_code} message={last_msg!r}")
            return None
        data = obj.get("data")
        if data is not None and str(data).strip():
            try:
                audio.extend(base64.b64decode(data))
            except Exception as e:
                logger.error(f"TTS base64 decode: {e}")
                return None
    if len(audio) == 0:
        logger.error(f"TTS empty audio (last code={last_code} message={last_msg!r})")
        return None
    return bytes(audio)


def text_to_speech(text: str, voice_type: str) -> Optional[bytes]:
    """Convert text to speech using Volcengine openspeech HTTP v3 unidirectional API."""
    api_key = _resolve_volc_api_key()
    if not api_key:
        raise ValueError(
            "Set VOLCENGINE_TTS_API_KEY or TTS_API_KEY "
            "(or VOLCENGINE_TTS_ACCESS_TOKEN for backward compatibility)"
        )

    endpoint = (
        os.getenv("VOLCENGINE_TTS_HTTP_ENDPOINT", "").strip() or VOLC_V3_DEFAULT_ENDPOINT
    ).rstrip("/")
    resource_id = _resource_id_for_request(voice_type)
    audio_format = _normalize_audio_format(os.getenv("VOLCENGINE_TTS_FORMAT", "mp3"))

    try:
        speed_ratio = float(os.getenv("VOLCENGINE_TTS_SPEED_RATIO", "1.2"))
    except ValueError:
        speed_ratio = 1.2

    payload = {
        "req_params": {
            "text": text.strip(),
            "speaker": voice_type.strip(),
            "additions": VOLC_V3_ADDITIONS,
            "audio_params": {
                "format": audio_format,
                "sample_rate": 24000,
                "speed_ratio": speed_ratio,
            },
        }
    }

    headers = {
        "x-api-key": api_key,
        "X-Api-Resource-Id": resource_id,
        "Connection": "keep-alive",
        "Content-Type": "application/json",
        "Accept": "*/*",
    }

    try:
        response = requests.post(
            endpoint,
            json=payload,
            headers=headers,
            timeout=180,
        )
        if response.status_code < 200 or response.status_code >= 300:
            logger.error(
                f"TTS HTTP {response.status_code}: {response.text[:4096]}"
            )
            return None
        return _parse_volc_v3_stream_body(response.text)
    except Exception as e:
        logger.error(f"TTS error: {e!s}")
        return None


def _process_line(args: tuple[int, ScriptLine, int]) -> tuple[int, Optional[bytes]]:
    """Process a single script line for TTS. Returns (index, audio_bytes)."""
    i, line, total = args

    # Select voice based on speaker gender (overridable via env, aligned with cmd/volc-tts defaults)
    if line.speaker == "male":
        voice_type = os.getenv(
            "VOLCENGINE_TTS_VOICE_MALE",
            "zh_male_yangguangqingnian_moon_bigtts",
        ).strip()
    else:
        voice_type = os.getenv(
            "VOLCENGINE_TTS_VOICE_FEMALE",
            "zh_female_sajiaonvyou_moon_bigtts",
        ).strip()

    logger.info(f"Processing line {i + 1}/{total} ({line.speaker})")
    audio = text_to_speech(line.paragraph, voice_type)

    if not audio:
        logger.warning(f"Failed to generate audio for line {i + 1}")

    return (i, audio)


def tts_node(script: Script, max_workers: int = 4) -> list[bytes]:
    """Convert script lines to audio chunks using TTS with multi-threading."""
    logger.info(f"Converting script to audio using {max_workers} workers...")

    total = len(script.lines)
    
    # Handle empty script case
    if total == 0:
        raise ValueError("Script contains no lines to process")

    if not _resolve_volc_api_key():
        raise ValueError(
            "Missing TTS credentials: set VOLCENGINE_TTS_API_KEY or TTS_API_KEY "
            "(or VOLCENGINE_TTS_ACCESS_TOKEN for backward compatibility)"
        )

    tasks = [(i, line, total) for i, line in enumerate(script.lines)]

    # Use ThreadPoolExecutor for parallel TTS generation
    results: dict[int, Optional[bytes]] = {}
    failed_indices: list[int] = []
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(_process_line, task): task[0] for task in tasks}
        for future in as_completed(futures):
            idx, audio = future.result()
            results[idx] = audio
            # Use `not audio` to catch both None and empty bytes
            if not audio:
                failed_indices.append(idx)

    # Log failed lines with 1-based indices for user-friendly output
    if failed_indices:
        logger.warning(
            f"Failed to generate audio for {len(failed_indices)}/{total} lines: "
            f"line numbers {sorted(i + 1 for i in failed_indices)}"
        )

    # Collect results in order, skipping failed ones
    audio_chunks = []
    for i in range(total):
        audio = results.get(i)
        if audio:
            audio_chunks.append(audio)

    logger.info(f"Generated {len(audio_chunks)}/{total} audio chunks successfully")
    
    if not audio_chunks:
        raise ValueError(
            f"TTS generation failed for all {total} lines. "
            "Check VOLCENGINE_TTS_API_KEY / TTS_API_KEY and VOLCENGINE_TTS_HTTP_ENDPOINT."
        )
    
    return audio_chunks


def mix_audio(audio_chunks: list[bytes]) -> bytes:
    """Combine audio chunks into a single audio file."""
    logger.info("Mixing audio chunks...")
    
    if not audio_chunks:
        raise ValueError("No audio chunks to mix - TTS generation may have failed")
    
    output = b"".join(audio_chunks)
    
    if len(output) == 0:
        raise ValueError("Mixed audio is empty - TTS generation may have failed")
    
    logger.info(f"Audio mixing complete: {len(output)} bytes")
    return output


def generate_markdown(script: Script, title: str = "Podcast Script") -> str:
    """Generate a markdown script from the podcast script."""
    lines = [f"# {title}", ""]

    for line in script.lines:
        speaker_name = "**Host (Male)**" if line.speaker == "male" else "**Host (Female)**"
        lines.append(f"{speaker_name}: {line.paragraph}")
        lines.append("")

    return "\n".join(lines)


def generate_podcast(
    script_file: str,
    output_file: str,
    transcript_file: Optional[str] = None,
) -> str:
    """Generate a podcast from a script JSON file."""

    # Read script JSON
    with open(script_file, "r", encoding="utf-8") as f:
        script_json = json.load(f)

    if "lines" not in script_json:
        raise ValueError(f"Invalid script format: missing 'lines' key. Got keys: {list(script_json.keys())}")

    script = Script.from_dict(script_json)
    logger.info(f"Loaded script with {len(script.lines)} lines")

    # Generate transcript markdown if requested
    if transcript_file:
        title = script_json.get("title", "Podcast Script")
        markdown_content = generate_markdown(script, title)
        transcript_dir = os.path.dirname(transcript_file)
        if transcript_dir:
            os.makedirs(transcript_dir, exist_ok=True)
        with open(transcript_file, "w", encoding="utf-8") as f:
            f.write(markdown_content)
        logger.info(f"Generated transcript to {transcript_file}")

    # Convert to audio
    audio_chunks = tts_node(script)

    if not audio_chunks:
        raise Exception("Failed to generate any audio")

    # Mix audio
    output_audio = mix_audio(audio_chunks)

    # Save output
    output_dir = os.path.dirname(output_file)
    if output_dir:
        os.makedirs(output_dir, exist_ok=True)
    with open(output_file, "wb") as f:
        f.write(output_audio)

    result = f"Successfully generated podcast to {output_file}"
    if transcript_file:
        result += f" and transcript to {transcript_file}"
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate podcast from script JSON file")
    parser.add_argument(
        "--script-file",
        required=True,
        help="Absolute path to script JSON file",
    )
    parser.add_argument(
        "--output-file",
        required=True,
        help="Output path for generated podcast MP3",
    )
    parser.add_argument(
        "--transcript-file",
        required=False,
        help="Output path for transcript markdown file (optional)",
    )

    args = parser.parse_args()

    try:
        result = generate_podcast(
            args.script_file,
            args.output_file,
            args.transcript_file,
        )
        print(result)
    except Exception as e:
        import traceback
        print(f"Error generating podcast: {e}")
        traceback.print_exc()
