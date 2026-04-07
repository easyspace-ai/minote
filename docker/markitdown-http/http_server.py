"""
HTTP API for MarkItDown (file upload + URL conversion).
Used by Notex when MARKITDOWN_URL points at this service instead of a local CLI.
"""

from __future__ import annotations

import os
from io import BytesIO

from fastapi import FastAPI, File, HTTPException, UploadFile
from pydantic import BaseModel

from markitdown import MarkItDown
from markitdown import StreamInfo

app = FastAPI(title="MarkItDown HTTP", version="1.0.0")

_enable_plugins = os.getenv("MARKITDOWN_ENABLE_PLUGINS", "false").lower() in (
    "1",
    "true",
    "yes",
)
_md = MarkItDown(enable_plugins=_enable_plugins)


class ConvertURLRequest(BaseModel):
    url: str


class ConvertResponse(BaseModel):
    text: str
    title: str | None = None


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/v1/convert", response_model=ConvertResponse)
async def convert_file(file: UploadFile = File(...)) -> ConvertResponse:
    if not file.filename:
        raise HTTPException(status_code=400, detail="missing filename")
    data = await file.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty file")
    ext = ""
    if "." in file.filename:
        ext = "." + file.filename.rsplit(".", 1)[-1].lower()
    stream_info = StreamInfo(filename=file.filename, extension=ext)
    try:
        result = _md.convert_stream(BytesIO(data), stream_info=stream_info)
    except Exception as e:
        raise HTTPException(status_code=422, detail=str(e)) from e
    return ConvertResponse(text=result.markdown, title=result.title)


@app.post("/v1/convert-url", response_model=ConvertResponse)
def convert_url(body: ConvertURLRequest) -> ConvertResponse:
    raw = (body.url or "").strip()
    if not raw:
        raise HTTPException(status_code=400, detail="missing url")
    try:
        result = _md.convert(raw)
    except Exception as e:
        raise HTTPException(status_code=422, detail=str(e)) from e
    return ConvertResponse(text=result.markdown, title=result.title)
