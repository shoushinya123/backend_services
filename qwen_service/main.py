#!/usr/bin/env python3
"""
Qwen-long-1M 模型服务
提供Token计数和文本生成接口
"""
import os
import sys
from typing import Optional
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import uvicorn

app = FastAPI(title="Qwen Model Service", version="1.0.0")

# 配置
QWEN_MODEL_PATH = os.getenv("QWEN_MODEL_PATH", "")
QWEN_API_KEY = os.getenv("QWEN_API_KEY", "")
QWEN_API_BASE = os.getenv("QWEN_API_BASE", "https://dashscope.aliyuncs.com/compatible-mode/v1")
LOCAL_MODE = os.getenv("QWEN_LOCAL_MODE", "true").lower() == "true"

# 全局模型实例（延迟加载）
_model = None
_tokenizer = None


class TokenCountRequest(BaseModel):
    text: str


class TokenCountResponse(BaseModel):
    token_count: int


class GenerateRequest(BaseModel):
    prompt: str
    max_tokens: Optional[int] = 2048
    temperature: Optional[float] = 0.7


class GenerateResponse(BaseModel):
    text: str
    token_count: Optional[int] = None


def load_local_model():
    """加载本地Qwen模型"""
    global _model, _tokenizer
    
    if _model is not None:
        return _model, _tokenizer
    
    try:
        from transformers import AutoModelForCausalLM, AutoTokenizer
        import torch
        
        model_path = QWEN_MODEL_PATH or "Qwen/Qwen2.5-7B-Instruct"
        
        print(f"Loading model from {model_path}...")
        _tokenizer = AutoTokenizer.from_pretrained(model_path, trust_remote_code=True)
        _model = AutoModelForCausalLM.from_pretrained(
            model_path,
            torch_dtype=torch.float16,
            device_map="auto",
            trust_remote_code=True
        )
        print("Model loaded successfully")
        
        return _model, _tokenizer
    except ImportError:
        print("Warning: transformers not installed, local mode disabled")
        return None, None
    except Exception as e:
        print(f"Warning: Failed to load local model: {e}")
        return None, None


def count_tokens_local(text: str) -> int:
    """使用本地tokenizer计算token数"""
    global _tokenizer
    
    if _tokenizer is None:
        _, _tokenizer = load_local_model()
    
    if _tokenizer is None:
        # Fallback: 简单估算
        return len(text) // 4
    
    try:
        tokens = _tokenizer.encode(text, add_special_tokens=False)
        return len(tokens)
    except Exception as e:
        print(f"Error counting tokens: {e}")
        return len(text) // 4


def count_tokens_api(text: str) -> int:
    """使用API计算token数"""
    try:
        import requests
        
        url = f"{QWEN_API_BASE}/tokenizers/estimate-token-count"
        headers = {
            "Authorization": f"Bearer {QWEN_API_KEY}",
            "Content-Type": "application/json"
        }
        data = {"text": text}
        
        response = requests.post(url, json=data, headers=headers, timeout=10)
        if response.status_code == 200:
            result = response.json()
            return result.get("token_count", len(text) // 4)
        else:
            print(f"API error: {response.status_code}")
            return len(text) // 4
    except Exception as e:
        print(f"Error calling API: {e}")
        return len(text) // 4


def generate_local(prompt: str, max_tokens: int = 2048, temperature: float = 0.7) -> str:
    """使用本地模型生成文本"""
    global _model, _tokenizer
    
    if _model is None or _tokenizer is None:
        _model, _tokenizer = load_local_model()
    
    if _model is None or _tokenizer is None:
        raise HTTPException(status_code=503, detail="Local model not available")
    
    try:
        import torch
        
        # 构建输入
        messages = [{"role": "user", "content": prompt}]
        text = _tokenizer.apply_chat_template(
            messages,
            tokenize=False,
            add_generation_prompt=True
        )
        model_inputs = _tokenizer([text], return_tensors="pt").to(_model.device)
        
        # 生成
        with torch.no_grad():
            generated_ids = _model.generate(
                **model_inputs,
                max_new_tokens=max_tokens,
                temperature=temperature,
                do_sample=True
            )
        
        generated_ids = [
            output_ids[len(input_ids):] for input_ids, output_ids in zip(model_inputs.input_ids, generated_ids)
        ]
        
        response = _tokenizer.batch_decode(generated_ids, skip_special_tokens=True)[0]
        return response
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Generation failed: {str(e)}")


def generate_api(prompt: str, max_tokens: int = 2048, temperature: float = 0.7) -> str:
    """使用API生成文本"""
    try:
        import requests
        
        url = f"{QWEN_API_BASE}/chat/completions"
        headers = {
            "Authorization": f"Bearer {QWEN_API_KEY}",
            "Content-Type": "application/json"
        }
        data = {
            "model": "qwen-long",
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": max_tokens,
            "temperature": temperature
        }
        
        response = requests.post(url, json=data, headers=headers, timeout=300)
        if response.status_code == 200:
            result = response.json()
            return result["choices"][0]["message"]["content"]
        else:
            raise HTTPException(status_code=response.status_code, detail=response.text)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"API call failed: {str(e)}")


@app.get("/health")
async def health():
    """健康检查"""
    return {"status": "healthy", "local_mode": LOCAL_MODE}


@app.post("/api/v1/token_count", response_model=TokenCountResponse)
async def token_count(request: TokenCountRequest):
    """计算token数量"""
    try:
        if LOCAL_MODE:
            count = count_tokens_local(request.text)
        else:
            count = count_tokens_api(request.text)
        
        return TokenCountResponse(token_count=count)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/generate", response_model=GenerateResponse)
async def generate(request: GenerateRequest):
    """生成文本"""
    try:
        if LOCAL_MODE:
            text = generate_local(request.prompt, request.max_tokens, request.temperature)
        else:
            text = generate_api(request.prompt, request.max_tokens, request.temperature)
        
        # 计算生成的token数
        token_count = None
        try:
            if LOCAL_MODE:
                token_count = count_tokens_local(text)
            else:
                token_count = count_tokens_api(text)
        except:
            pass
        
        return GenerateResponse(text=text, token_count=token_count)
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8004"))
    uvicorn.run(app, host="0.0.0.0", port=port)

