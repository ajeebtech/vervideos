#!/usr/bin/env python3
"""
Parse After Effects .aepx files to extract asset references.
"""

import json
import os
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import List, Dict, Set


def parse_aepx(aepx_path: str) -> Dict:
    """
    Parse an .aepx file and extract all asset references.
    
    Returns a dictionary with:
    - project_file: path to the .aepx file
    - assets: list of asset file paths found
    - missing_assets: list of referenced but missing files
    """
    result = {
        "project_file": os.path.abspath(aepx_path),
        "assets": [],
        "missing_assets": [],
        "total_size": 0
    }
    
    try:
        tree = ET.parse(aepx_path)
        root = tree.getroot()
    except Exception as e:
        # Print to stderr to avoid breaking JSON output
        print(f"Warning: XML parse error: {e}", file=sys.stderr)
        # Return empty result on parse error
        return result
    
    # Set to avoid duplicates
    asset_paths: Set[str] = set()
    
    # Look for file references in various elements
    # After Effects stores file paths in different elements depending on the asset type
    
    # Method 1: Look for <fileReference> elements with fullpath attribute (most common in .aepx)
    # Handle both namespaced and non-namespaced elements
    for file_ref in root.iter():
        # Check if this is a fileReference element (with or without namespace)
        if file_ref.tag.endswith('fileReference') or 'fileReference' in file_ref.tag:
            if 'fullpath' in file_ref.attrib:
                fullpath = file_ref.attrib['fullpath']
                if fullpath and fullpath.strip():
                    asset_paths.add(fullpath.strip())
    
    # Method 2: Look for <fullpath> elements (text content)
    for fullpath in root.iter('fullpath'):
        if fullpath.text:
            asset_paths.add(fullpath.text)
    
    # Method 3: Look for file paths in specific asset elements
    for elem in root.iter():
        # Check for 'file' attributes or text content that looks like paths
        if elem.tag in ['file', 'path', 'src', 'source']:
            if elem.text:
                asset_paths.add(elem.text)
            for attr, value in elem.attrib.items():
                if 'path' in attr.lower() or 'file' in attr.lower():
                    asset_paths.add(value)
    
    # Get the project directory to resolve relative paths
    project_dir = os.path.dirname(os.path.abspath(aepx_path))
    
    # Process each asset path
    for asset_path in asset_paths:
        if not asset_path or asset_path.strip() == '':
            continue
            
        # Clean up the path
        asset_path = asset_path.strip()
        
        # Skip if it's a system path or URL
        if asset_path.startswith(('http://', 'https://', 'file://')):
            continue
        
        # Convert to absolute path if relative
        if not os.path.isabs(asset_path):
            asset_path = os.path.join(project_dir, asset_path)
        
        # Normalize the path
        asset_path = os.path.normpath(asset_path)
        
        # Check if file exists
        if os.path.exists(asset_path) and os.path.isfile(asset_path):
            file_size = os.path.getsize(asset_path)
            result["assets"].append({
                "path": asset_path,
                "relative_path": os.path.relpath(asset_path, project_dir),
                "filename": os.path.basename(asset_path),
                "extension": os.path.splitext(asset_path)[1],
                "size": file_size
            })
            result["total_size"] += file_size
        else:
            result["missing_assets"].append(asset_path)
    
    # Sort assets by path for consistency
    result["assets"].sort(key=lambda x: x["path"])
    result["missing_assets"].sort()
    
    return result


def main():
    if len(sys.argv) != 2:
        print("Usage: parse_aepx.py <path-to-aepx-file>", file=sys.stderr)
        sys.exit(1)
    
    aepx_path = sys.argv[1]
    
    if not os.path.exists(aepx_path):
        print(f"Error: File '{aepx_path}' not found", file=sys.stderr)
        sys.exit(1)
    
    if not aepx_path.endswith('.aepx'):
        print(f"Error: File must have .aepx extension", file=sys.stderr)
        sys.exit(1)
    
    result = parse_aepx(aepx_path)
    
    # Output as JSON
    print(json.dumps(result, indent=2))
    
    # Exit with error code if there are missing assets
    if result["missing_assets"]:
        sys.exit(2)


if __name__ == "__main__":
    main()

