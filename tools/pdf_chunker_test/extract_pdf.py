#!/usr/bin/env python3
"""提取PDF文本"""
import sys

def extract_with_pypdf2(pdf_path):
    try:
        import PyPDF2
        text = ""
        with open(pdf_path, 'rb') as f:
            reader = PyPDF2.PdfReader(f)
            for page in reader.pages:
                page_text = page.extract_text()
                if page_text:
                    text += page_text + "\n"
        return text if text.strip() else None
    except Exception as e:
        return None

def extract_with_pdfplumber(pdf_path):
    try:
        import pdfplumber
        text = ""
        with pdfplumber.open(pdf_path) as pdf:
            for page in pdf.pages:
                page_text = page.extract_text()
                if page_text:
                    text += page_text + "\n"
        return text if text.strip() else None
    except Exception as e:
        return None

def extract_with_pypdf(pdf_path):
    try:
        import pypdf
        text = ""
        with open(pdf_path, 'rb') as f:
            reader = pypdf.PdfReader(f)
            for page in reader.pages:
                page_text = page.extract_text()
                if page_text:
                    text += page_text + "\n"
        return text if text.strip() else None
    except Exception as e:
        return None

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 extract_pdf.py <pdf_file>")
        sys.exit(1)
    
    pdf_path = sys.argv[1]
    
    # 尝试不同的PDF库
    text = None
    for extractor in [extract_with_pdfplumber, extract_with_pypdf, extract_with_pypdf2]:
        try:
            result = extractor(pdf_path)
            if isinstance(result, tuple) and result[0] is None:
                continue
            text = result
            break
        except:
            continue
    
    if text is None:
        print("错误: 无法提取PDF文本，请安装pdfplumber或PyPDF2", file=sys.stderr)
        sys.exit(1)
    
    print(text, end='')

