#!/usr/bin/env bash
set -euo pipefail

if [ -f "PhotoArchiverMac.xcodeproj/project.pbxproj" ]; then
  open "PhotoArchiverMac.xcodeproj"
else
  open Package.swift
fi
