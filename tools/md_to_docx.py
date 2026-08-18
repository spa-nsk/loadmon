#!/usr/bin/env python3
"""Convert loadmon markdown UI doc to DOCX."""

import re
import sys
from pathlib import Path

from docx import Document
from docx.enum.text import WD_PARAGRAPH_ALIGNMENT
from docx.oxml.ns import qn
from docx.shared import Cm, Pt
from docx.enum.table import WD_TABLE_ALIGNMENT


def strip_md_inline(text: str) -> str:
    text = re.sub(r"\*\*(.+?)\*\*", r"\1", text)
    text = re.sub(r"`([^`]+)`", r"\1", text)
    text = re.sub(r"\*(.+?)\*", r"\1", text)
    return text


def add_rich_paragraph(doc, text: str, style=None):
    p = doc.add_paragraph(style=style)
    parts = re.split(r"(\*\*.+?\*\*|`[^`]+`)", text)
    for part in parts:
        if not part:
            continue
        if part.startswith("**") and part.endswith("**"):
            run = p.add_run(part[2:-2])
            run.bold = True
        elif part.startswith("`") and part.endswith("`"):
            run = p.add_run(part[1:-1])
            run.font.name = "Consolas"
            run._element.rPr.rFonts.set(qn("w:eastAsia"), "Consolas")
        else:
            p.add_run(part)
    return p


def is_table_row(line: str) -> bool:
    s = line.strip()
    return s.startswith("|") and s.endswith("|") and "|" in s[1:-1]


def is_separator_row(line: str) -> bool:
    return bool(re.match(r"^\|\s*[-:| ]+\|\s*$", line.strip()))


def parse_table_rows(lines: list[str]) -> list[list[str]]:
    rows = []
    for line in lines:
        if is_separator_row(line):
            continue
        cells = [strip_md_inline(c.strip()) for c in line.strip().strip("|").split("|")]
        rows.append(cells)
    return rows


def add_table(doc, rows: list[list[str]]):
    if not rows:
        return
    cols = max(len(r) for r in rows)
    table = doc.add_table(rows=len(rows), cols=cols)
    table.style = "Table Grid"
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    for i, row in enumerate(rows):
        for j in range(cols):
            cell = table.rows[i].cells[j]
            cell.text = row[j] if j < len(row) else ""
            for paragraph in cell.paragraphs:
                for run in paragraph.runs:
                    run.font.size = Pt(10)
                if i == 0:
                    for run in paragraph.runs:
                        run.bold = True


def convert(md_path: Path, docx_path: Path):
    lines = md_path.read_text(encoding="utf-8").splitlines()
    doc = Document()

    section = doc.sections[0]
    section.top_margin = Cm(2)
    section.bottom_margin = Cm(2)
    section.left_margin = Cm(2.5)
    section.right_margin = Cm(2)

    style = doc.styles["Normal"]
    style.font.name = "Calibri"
    style.font.size = Pt(11)
    style._element.rPr.rFonts.set(qn("w:eastAsia"), "Calibri")

    i = 0
    in_code = False
    code_lines: list[str] = []

    while i < len(lines):
        line = lines[i]

        if line.strip().startswith("```"):
            if in_code:
                p = doc.add_paragraph("\n".join(code_lines))
                p.style = "Intense Quote"
                for run in p.runs:
                    run.font.name = "Consolas"
                    run.font.size = Pt(9)
                    run._element.rPr.rFonts.set(qn("w:eastAsia"), "Consolas")
                code_lines = []
                in_code = False
            else:
                in_code = True
            i += 1
            continue

        if in_code:
            code_lines.append(line)
            i += 1
            continue

        if line.strip() == "---":
            doc.add_paragraph()
            i += 1
            continue

        if line.startswith("# "):
            doc.add_heading(strip_md_inline(line[2:].strip()), level=0)
            i += 1
            continue
        if line.startswith("## "):
            doc.add_heading(strip_md_inline(line[3:].strip()), level=1)
            i += 1
            continue
        if line.startswith("### "):
            doc.add_heading(strip_md_inline(line[4:].strip()), level=2)
            i += 1
            continue

        if is_table_row(line):
            table_lines = []
            while i < len(lines) and is_table_row(lines[i]):
                table_lines.append(lines[i])
                i += 1
            add_table(doc, parse_table_rows(table_lines))
            doc.add_paragraph()
            continue

        if line.strip().startswith("- "):
            while i < len(lines) and lines[i].strip().startswith("- "):
                add_rich_paragraph(doc, lines[i].strip()[2:], style="List Bullet")
                i += 1
            continue

        if line.strip().startswith("*") and line.strip().endswith("*") and not line.strip().startswith("**"):
            p = doc.add_paragraph(strip_md_inline(line.strip().strip("*")))
            p.alignment = WD_PARAGRAPH_ALIGNMENT.CENTER
            for run in p.runs:
                run.italic = True
            i += 1
            continue

        if line.strip():
            add_rich_paragraph(doc, line.strip())
        i += 1

    doc.save(docx_path)
    print(f"Written: {docx_path}")


if __name__ == "__main__":
    root = Path(__file__).resolve().parents[1]
    md = Path(sys.argv[1]) if len(sys.argv) > 1 else root / "loadmon_ui.md"
    out = Path(sys.argv[2]) if len(sys.argv) > 2 else root / "loadmon_ui.docx"
    convert(md, out)
