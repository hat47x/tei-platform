#!/usr/bin/env python3
"""Compatibility entrypoint for historical diagnostic-slice commands."""
from pathlib import Path
import runpy
import sys

if __name__ == '__main__':
    sys.argv.insert(1, '--diagnostic-slices')
    runpy.run_path(str(Path(__file__).with_name('verify-order-workflow.py')), run_name='__main__')
