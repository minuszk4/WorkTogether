import sys
import os
import re
import subprocess

if len(sys.argv) < 3:
    print("usage: python task_brief.py PLAN_FILE TASK_NUMBER [OUTFILE]")
    sys.exit(2)

plan_file = sys.argv[1]
task_num = sys.argv[2]

if len(sys.argv) >= 4:
    outfile = sys.argv[3]
else:
    root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"]).decode("utf-8").strip()
    outdir = os.path.join(root, ".superpowers", "sdd")
    os.makedirs(outdir, exist_ok=True)
    with open(os.path.join(outdir, ".gitignore"), "w") as f:
        f.write("*\n")
    outfile = os.path.join(outdir, f"task-{task_num}-brief.md")

infence = False
intask = False
task_pattern = re.compile(rf"^#+[ \t]+Task[ \t]+{task_num}([^0-9]|$)")
general_task_pattern = re.compile(r"^#+[ \t]+Task[ \t]+[0-9]+")

brief_lines = []

with open(plan_file, "r", encoding="utf-8") as f:
    for line in f:
        if line.startswith("```"):
            infence = not infence
        
        if not infence:
            if general_task_pattern.match(line):
                if task_pattern.match(line):
                    intask = True
                else:
                    intask = False
        
        if intask:
            brief_lines.append(line)

if not brief_lines:
    print(f"task {task_num} not found in {plan_file}", file=sys.stderr)
    sys.exit(3)

with open(outfile, "w", encoding="utf-8") as f:
    f.writelines(brief_lines)

print(f"wrote {outfile}: {len(brief_lines)} lines")
