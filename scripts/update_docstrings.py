#!/usr/bin/env python3
"""
Docstring Updater for Mimir-AIP

This script scans all Python files under src/ and inserts:
- A module-level docstring if missing.
- A placeholder PEP-257 docstring for any class or function missing one.
"""
import ast
import os

MODULE_TEMPLATE = '"""\nModule {file}: TODO description.\n"""'

def process_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        lines = f.read().splitlines()
    tree = ast.parse("\n".join(lines))
    modified = False
    # Module docstring
    if not ast.get_docstring(tree):
        lines.insert(0, MODULE_TEMPLATE.format(file=os.path.basename(filepath)))
        modified = True
    # Class and function docstrings
    for node in ast.walk(tree):
        if isinstance(node, (ast.ClassDef, ast.FunctionDef)):
            if not ast.get_docstring(node):
                indent = ' ' * node.col_offset
                doc = f'{indent}"""{node.name}: TODO add description."""'
                insert_at = node.body[0].lineno - 1 if node.body else node.lineno
                lines.insert(insert_at, doc)
                modified = True
    if modified:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write("\n".join(lines))
        print(f"Updated docstrings in {filepath}")

def main():
    root = os.path.join(os.path.dirname(__file__), '..', 'src')
    for dirpath, _, filenames in os.walk(root):
        for fname in filenames:
            if fname.endswith('.py'):
                process_file(os.path.join(dirpath, fname))

if __name__ == '__main__':
    main()
