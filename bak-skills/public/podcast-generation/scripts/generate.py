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


def _volc_resource_id_for_voice(voice: str) -> str:
    v = (voice or "").strip()
    low = v.lower()
    if v.startswith("S_"):
        return "volc.megatts.default"
    if low.startswith("seed-tts") or "seed_tts" in low:
        return v
    return "volc.service_type.10029"


def text_to_speech(text: str, voice_type: str) -> Optional[bytes]:
    """Convert text to speech via Volcengine openspeech HTTP unidirectional (same as gateway /api/tts)."""
    api_key = (os.getenv("VOLCENGINE_TTS_API_KEY") or os.getenv("TTS_API_KEY") or "").strip()
    token = (os.getenv("VOLCENGINE_TTS_ACCESS_TOKEN") or "").strip()
    if not api_key and token:
        api_key = token
    app_id = (os.getenv("VOLCENGINE_TTS_APP_ID") or "").strip()
    if not api_key and app_id.lower().startswith("api-key-"):
        api_key = app_id

    if not api_key:
        raise ValueError(
            "Set VOLCENGINE_TTS_API_KEY or TTS_API_KEY (openspeech x-api-key); "
            "VOLCENGINE_TTS_ACCESS_TOKEN is accepted as a legacy alias."
        )

    endpoint = (os.getenv("VOLCENGINE_TTS_HTTP_ENDPOINT") or "").strip().rstrip("/")
    if not endpoint:
        endpoint = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"

    resource_id = (os.getenv("VOLCENGINE_TTS_RESOURCE_ID") or "").strip()
    if not resource_id:
        resource_id = _volc_resource_id_for_voice(voice_type)

    additions = (
        '{"disable_markdown_filter":true,"enable_language_detector":true,"enable_latex_tn":true,'
        '"disable_default_bit_rate":true,"max_length_to_filter_parenthesis":0,'
        '"cache_config":{"text_type":1,"use_cache":true}}'
    )
    payload = {
        "req_params": {
            "text": text.strip(),
            "speaker": voice_type.strip(),
            "additions": additions,
            "audio_params": {"format": "mp3", "sample_rate": 24000},
        }
    }
    headers = {
        "Content-Type": "application/json",
        "x-api-key": api_key,
        "X-Api-Resource-Id": resource_id,
        "Connection": "keep-alive",
        "Accept": "*/*",
    }

    try:
        response = requests.post(endpoint, json=payload, headers=headers, timeout=180, stream=True)

        if response.status_code != 200:
            logger.error(f"TTS API error: {response.status_code} - {response.text[:2048]}")
            return None

        audio_buf = bytearray()
        for raw_line in response.iter_lines(decode_unicode=True):
            if not raw_line:
                continue
            try:
                evt = json.loads(raw_line)
            except json.JSONDecodeError:
                continue
            code = evt.get("code", 0)
            if code not in (0, 20000000):
                msg = (evt.get("message") or "unknown error").strip()
                logger.error(f"TTS volc error code={code} message={msg}")
                return None
            data = evt.get("data")
            if data and str(data).strip():
                try:
                    audio_buf.extend(base64.b64decode(str(data)))
                except Exception as e:  # noqa: BLE001
                    logger.error(f"TTS base64 decode: {e}")
                    return None

        if not audio_buf:
            logger.error("TTS empty audio stream")
            return None
        return bytes(audio_buf)

    except Exception as e:
        logger.error(f"TTS error: {str(e)}")
        return None


def _process_line(args: tuple[int, ScriptLine, int]) -> tuple[int, Optional[bytes]]:
    """Process a single script line for TTS. Returns (index, audio_bytes)."""
    i, line, total = args

    # Select voice based on speaker gender
    if line.speaker == "male":
        voice_type = "zh_male_yangguangqingnian_moon_bigtts"  # Male voice
    else:
        voice_type = "zh_female_sajiaonvyou_moon_bigtts"  # Female voice

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

    if not (
        os.getenv("VOLCENGINE_TTS_API_KEY")
        or os.getenv("TTS_API_KEY")
        or os.getenv("VOLCENGINE_TTS_ACCESS_TOKEN")
    ):
        raise ValueError(
            "Missing TTS credentials: set VOLCENGINE_TTS_API_KEY or TTS_API_KEY (or VOLCENGINE_TTS_ACCESS_TOKEN as legacy alias)"
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
            "Please check VOLCENGINE_TTS_API_KEY / TTS_API_KEY environment variables."
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
