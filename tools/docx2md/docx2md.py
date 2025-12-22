#!/usr/bin/env python3
"""
DOCX to Markdown converter
"""
import sys
import os
import argparse
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

def extract_text_from_docx(docx_path):
    """Extract text from DOCX file"""
    try:
        # Try using python-docx if available
        try:
            import docx
            doc = docx.Document(docx_path)
            text_parts = []
            
            for para in doc.paragraphs:
                text = para.text.strip()
                if not text:
                    continue
                
                # Check if it's a heading
                try:
                    style_name = para.style.name if para.style else None
                    if style_name and style_name.startswith('Heading'):
                        level = 1
                        if 'Heading 1' in style_name or 'Title' in style_name:
                            level = 1
                        elif 'Heading 2' in style_name:
                            level = 2
                        elif 'Heading 3' in style_name:
                            level = 3
                        elif 'Heading 4' in style_name:
                            level = 4
                        else:
                            level = 2
                        text_parts.append('#' * level + ' ' + text + '\n\n')
                    else:
                        text_parts.append(text + '\n\n')
                except:
                    # If style check fails, treat as normal paragraph
                    text_parts.append(text + '\n\n')
            
            return ''.join(text_parts)
        except ImportError:
            pass
        
        # Fallback: extract from XML directly
        with zipfile.ZipFile(docx_path, 'r') as docx:
            # Read the main document XML
            xml_content = docx.read('word/document.xml')
            root = ET.fromstring(xml_content)
            
            # Define namespaces
            ns = {'w': 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'}
            
            text_parts = []
            for para in root.findall('.//w:p', ns):
                para_text = []
                for t in para.findall('.//w:t', ns):
                    if t.text:
                        para_text.append(t.text)
                
                text = ''.join(para_text).strip()
                if text:
                    text_parts.append(text)
            
            return '\n\n'.join(text_parts)
            
    except Exception as e:
        return f"Error extracting text: {str(e)}"

def convert_docx_to_markdown(input_path, output_path=None):
    """Convert DOCX to Markdown"""
    if not os.path.exists(input_path):
        print(f"Error: Input file not found: {input_path}", file=sys.stderr)
        return False
    
    if output_path is None:
        base = os.path.splitext(input_path)[0]
        output_path = base + '.md'
    
    # Extract text
    text = extract_text_from_docx(input_path)
    
    # Add header
    title = Path(input_path).stem.replace('_', ' ').replace('-', ' ')
    markdown = f"# {title}\n\n"
    markdown += f"> 从DOCX转换: {os.path.basename(input_path)}\n\n"
    markdown += "---\n\n"
    markdown += text
    
    # Write output
    try:
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(markdown)
        print(f"✅ DOCX转换成功: {input_path} -> {output_path}")
        return True
    except Exception as e:
        print(f"Error writing output: {e}", file=sys.stderr)
        return False

def main():
    parser = argparse.ArgumentParser(description='Convert DOCX to Markdown')
    parser.add_argument('-input', required=True, help='Input DOCX file path')
    parser.add_argument('-output', help='Output Markdown file path (optional)')
    
    args = parser.parse_args()
    
    success = convert_docx_to_markdown(args.input, args.output)
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()

