#!/usr/bin/env python3
"""Shared workflow/schema versions for the PPT agent harness (v4)."""

from __future__ import annotations

from typing import Any


WORKFLOW_VERSION = "2026.03.31-v4"
RESEARCH_SCHEMA_VERSION = "4.0"
OUTLINE_SCHEMA_VERSION = "4.0"
PLANNING_SCHEMA_VERSION = "4.0"
PLANNING_PACKET_VERSION = "4.0"
PLANNING_CONTINUITY_VERSION = "4.0"
DISPATCH_PLAN_VERSION = "4.0"
HTML_PACKET_VERSION = "4.0"

DELIVERY_MANIFEST_VERSION = "4.0"


def build_workflow_metadata(stage: str, **extra: Any) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "stage": stage,
        "workflow_version": WORKFLOW_VERSION,
        "research_schema_version": RESEARCH_SCHEMA_VERSION,
        "outline_schema_version": OUTLINE_SCHEMA_VERSION,
        "planning_schema_version": PLANNING_SCHEMA_VERSION,
        "planning_packet_version": PLANNING_PACKET_VERSION,
        "planning_continuity_version": PLANNING_CONTINUITY_VERSION,
        "dispatch_plan_version": DISPATCH_PLAN_VERSION,
        "html_packet_version": HTML_PACKET_VERSION,
        "delivery_manifest_version": DELIVERY_MANIFEST_VERSION,
    }
    payload.update({key: value for key, value in extra.items() if value is not None})
    return payload
