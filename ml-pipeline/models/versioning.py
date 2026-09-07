"""Model version tagging for prod vs demo artifacts."""

from __future__ import annotations

from datetime import date


def prod_version(model_name: str) -> str:
    return f"prod-{model_name}-{date.today().isoformat()}"


def demo_version(model_name: str) -> str:
    return f"demo-{model_name}-synthetic"


def tag_artifact(artifact: dict, version: str) -> dict:
    artifact["model_version"] = version
    return artifact
