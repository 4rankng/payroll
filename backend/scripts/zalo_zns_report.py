#!/usr/bin/env python3
"""Customers whose phone is NOT linked to Zalo (ZNS error -118), from prod (read-only).

Usage: ~/.claude/skills/.venv/bin/python3 backend/scripts/zalo_zns_report.py
Writes: ~/Downloads/zalo-not-linked-<timestamp>.xlsx — one sheet: Tên, SĐT.
A phone counts as not linked only if it never received a ZNS (some failed -118
attempts may just mean the customer linked Zalo later).
"""
import subprocess, sys
from datetime import datetime

from openpyxl import Workbook
from openpyxl.styles import Font

PROD = "root@tingting.vip"
CONTAINER = "payroll-mysql"
DB = "payroll_db"

SQL = ("SELECT n.phone, COALESCE(NULLIF(n.customer_name, ''), e.fullname) AS ten, "
       "COALESCE(p.name, '') AS duan, n.status, n.provider_code "
       "FROM flexpay_salary_notifications n "
       "LEFT JOIN employees e ON e.id = n.employee_id "
       "LEFT JOIN projects p ON p.id = n.project_id")

def mysql(sql):
    remote = (f"docker exec -i {CONTAINER} sh -c "
              f"'exec env MYSQL_PWD=\"$MYSQL_ROOT_PASSWORD\" mysql "
              f"--default-character-set=utf8mb4 -uroot --batch -D {DB}'")
    p = subprocess.run(["ssh", "-o", "ConnectTimeout=15", PROD, remote],
                       input=sql.encode(), capture_output=True)
    if p.returncode != 0:
        sys.exit(f"ERROR: query failed:\n{p.stderr.decode(errors='replace')}")
    lines = [l for l in p.stdout.decode("utf-8", errors="replace").splitlines() if l]
    header = [h.lower() for h in lines[0].split("\t")]
    return [dict(zip(header, l.split("\t"))) for l in lines[1:]]

def main():
    rows = mysql(SQL)
    print(f"Rows: {len(rows)} notifications")

    phones = {}
    for r in rows:
        p = phones.setdefault(r["phone"], {"ten": "", "duan": set(), "sent": False, "not_linked": False})
        if not p["ten"]:
            p["ten"] = r["ten"] or ""
        if r["duan"]:
            p["duan"].add(r["duan"])
        if r["status"] == "sent":
            p["sent"] = True
        try:
            if int(r["provider_code"]) == -118:
                p["not_linked"] = True
        except (ValueError, TypeError):
            pass

    not_linked = sorted(
        ([p["ten"], phone, ", ".join(sorted(p["duan"]))]
         for phone, p in phones.items() if p["not_linked"] and not p["sent"]),
        key=lambda x: x[0].lower())

    wb = Workbook()
    ws = wb.active
    ws.title = "Chưa liên kết Zalo"
    ws.append(["Tên", "SĐT", "Dự án"])
    for c in ws[1]:
        c.font = Font(bold=True)
    for r in not_linked:
        ws.append(r)
    ws.column_dimensions["A"].width = 30
    ws.column_dimensions["B"].width = 15
    ws.column_dimensions["C"].width = 30
    ws.freeze_panes = "A2"

    out = f"/Users/dev/Downloads/zalo-not-linked-{datetime.now():%Y%m%d-%H%M}.xlsx"
    wb.save(out)
    bad = [r for r in not_linked if "?" in r[0]]
    print(f"Saved: {out}")
    print(f"Chưa liên kết Zalo: {len(not_linked)} customers | rows with '?' in name: {len(bad)}")

if __name__ == "__main__":
    main()
