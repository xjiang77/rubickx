from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any, Protocol


class PatternError(Exception):
    def __init__(self, code: str):
        super().__init__(code)
        self.code = code


class Formatter(Protocol):
    media_type: str
    def render(self, records: list[dict[str, str]]) -> Any: ...


class CsvFormatter:
    media_type = "text/csv"

    def render(self, records):
        rows = ["id,name"]
        rows.extend(f"{record['id']},{record['name']}" for record in records)
        return "\n".join(rows)


class JsonFormatter:
    media_type = "application/json"

    def render(self, records):
        return [dict(record) for record in records]


class ExportJob(ABC):
    @abstractmethod
    def create_formatter(self) -> Formatter: ...

    def export(self, records):
        formatter = self.create_formatter()
        return {"media_type": formatter.media_type, "body": formatter.render(records)}


class CsvExportJob(ExportJob):
    def create_formatter(self):
        return CsvFormatter()


class JsonExportJob(ExportJob):
    def create_formatter(self):
        return JsonFormatter()


def evaluate(input_data):
    creators = {"csv": CsvExportJob, "json": JsonExportJob}
    creator_type = creators.get(input_data.get("format"))
    if creator_type is None:
        raise PatternError("unsupported_format")
    return creator_type().export(input_data.get("records", []))
