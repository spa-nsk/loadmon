#!/usr/bin/env python3
"""Split Go source files by line ranges into multiple files in the same package."""

from pathlib import Path


def read_header(lines: list[str]) -> tuple[int, list[str]]:
    """Return (body_start_index, header_lines: doc + package + imports)."""
    i = 0
    header: list[str] = []
    while i < len(lines) and not lines[i].startswith("package "):
        header.append(lines[i])
        i += 1
    if i >= len(lines):
        raise ValueError("package declaration not found")
    header.append(lines[i])
    i += 1
    while i < len(lines) and lines[i].strip() == "":
        header.append(lines[i])
        i += 1
    if i < len(lines) and lines[i].startswith("import"):
        if lines[i].strip() == "import (":
            header.append(lines[i])
            i += 1
            while i < len(lines):
                header.append(lines[i])
                if lines[i].strip() == ")":
                    i += 1
                    break
                i += 1
        else:
            header.append(lines[i])
            i += 1
    while i < len(lines) and lines[i].strip() == "":
        header.append(lines[i])
        i += 1
    return i, header


def write_split(src: Path, outputs: dict[str, list[tuple[int, int]]], delete_src: bool = True) -> None:
    lines = src.read_text(encoding="utf-8").splitlines(keepends=True)
    body_start, header = read_header(lines)

    for out_name, ranges in outputs.items():
        out_path = src.parent / out_name
        chunks: list[str] = []
        for start, end in ranges:
            s = max(body_start, start - 1)
            e = len(lines) if end == -1 else min(len(lines), end)
            chunks.extend(lines[s:e])
        body = "".join(chunks).lstrip("\n")
        if body and not body.endswith("\n"):
            body += "\n"
        out_path.write_text("".join(header) + ("\n" + body if body else ""), encoding="utf-8")
        print(f"  wrote {out_path.name} ({len(body.splitlines())} lines)")

    if delete_src:
        src.unlink()
        print(f"  removed {src.name}")


def main() -> None:
    root = Path(__file__).resolve().parents[1]

    splits: dict[Path, dict[str, list[tuple[int, int]]]] = {
        # --- controllers ---
        root / "controllers/default.go": {
            "main_controller.go": [(36, 429), (1494, 1617)],
            "report_calc.go": [(430, 741)],
            "report_charts.go": [(742, 1493)],
            "error_controller.go": [(1618, -1)],
        },
        # --- cmdb ---
        root / "cmdb/savemetricperf.go": {
            "savemetric_orchestration.go": [(17, 276)],
            "savestorage.go": [(277, 393)],
            "saveclusterrecord.go": [(394, 437)],
            "savehostrecord.go": [(438, 647)],
            "savevmrecord.go": [(648, -1)],
        },
        # --- grpcclient ---
        root / "grpcclient/clientfunc.go": {
            "dial.go": [(38, 91)],
            "client_vcenter.go": [(92, 193)],
            "client_cluster.go": [(194, 401)],
            "client_host.go": [(402, 532)],
            "client_vm.go": [(561, 646), (852, 964)],
            "client_vm_convert.go": [(647, 851)],
            "client_datastore.go": [(965, 1051)],
            "client_metric_clear.go": [(1052, 1107)],
            "client_stubs.go": [(1109, 1124)],
            "client_oneview.go": [(1126, 1223)],
            "client_db.go": [(1225, 1294)],
            "client_registration.go": [(1296, -1)],
        },
        root / "grpcclient/message_cli.go": {
            "message_vsphere.go": [(18, 410)],
            "message_spr.go": [(411, 519), (1151, 1186)],
            "message_oneview.go": [(520, 1150)],
            "message_db_record.go": [(1187, 1600)],
            "message_db_perf.go": [(1601, -1)],
        },
        # --- grpcserver ---
        root / "grpcserver/serverfunc.go": {
            "server.go": [(38, 114)],
            "handler_vcenter.go": [(116, 220)],
            "handler_cluster.go": [(221, 457), (635, 684), (703, 751), (1654, 1716), (1831, 1886), (1961, 1996)],
            "copy_cluster.go": [(529, 634), (686, 702), (752, 809), (810, 827)],
            "handler_host.go": [(828, 889), (1591, 1653), (1717, 1960)],
            "copy_host.go": [(458, 528), (890, 968)],
            "handler_vm.go": [(969, 1590)],
            "copy_vm.go": [(1018, 1028)],
            "handler_datastore.go": [(1997, 2103)],
            "handler_metrics_clear.go": [(2104, 2444)],
            "handler_hardware.go": [(2445, 2618)],
            "handler_db_clusterspr.go": [(2619, 2788)],
            "handler_endpoint.go": [(2789, -1)],
        },
        root / "grpcserver/message_srv.go": {
            "message_vm.go": [(18, 225)],
            "message_host.go": [(226, 668)],
            "message_cluster.go": [(669, 906)],
            "message_spr.go": [(907, 1016), (1650, 1687)],
            "message_hardware.go": [(1017, 1649)],
            "message_db_records.go": [(1688, -1)],
        },
    }

    for src, outputs in splits.items():
        if not src.exists():
            print(f"skip missing {src}")
            continue
        print(f"split {src.relative_to(root)}:")
        write_split(src, outputs)


if __name__ == "__main__":
    main()
