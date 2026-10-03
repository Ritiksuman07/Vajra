#!/usr/bin/env python3
import os, sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(__file__))))

from flask import Flask, render_template, request, jsonify

app = Flask(__name__,
            template_folder="templates",
            static_folder="static")

@app.route("/")
def index():
    return render_template("index.html")

@app.route("/agents")
def agents():
    return render_template("agents.html")

@app.route("/chat")
def chat():
    return render_template("chat.html")

@app.route("/connect")
def connect():
    return render_template("connect.html")

@app.route("/verification")
def verification():
    return render_template("verification.html")

@app.route("/api/health")
def health():
    return jsonify({"status":"ok","service":"vajra","version":"1.0.0"})

@app.route("/api/agents")
def api_agents():
    return jsonify({
        "agents":[
            {"id":"chief-1","role":"chief","status":"running","model":"llama3.1:70b","pod":"running"},
            {"id":"coder-1","role":"coder","status":"running","model":"deepseek-coder","pod":"running"},
            {"id":"researcher-1","role":"researcher","status":"idle","model":"llama3.1:8b"},
            {"id":"executor-1","role":"executor","status":"idle","model":"phi-3.5-mini"},
            {"id":"reviewer-1","role":"reviewer","status":"idle","model":"llama3.1:8b"},
        ],
        "count":5,
        "status":"active"
    })

@app.route("/api/tasks", methods=["POST"])
def api_tasks():
    data = request.get_json() or {}
    return jsonify({
        "task_id":"task-"+str(os.urandom(4).hex()),
        "status":"submitted",
        "message":"Task submitted to Vajra agent team",
        "description": data.get("task",""),
    })

@app.route("/api/models")
def api_models():
    return jsonify({
        "models":[
            {"name":"llama3.1:70b","provider":"ollama","type":"local","status":"available","size":"40GB","context":131072},
            {"name":"llama3.1:8b","provider":"ollama","type":"local","status":"available","size":"4.7GB","context":131072},
            {"name":"deepseek-coder-v2","provider":"ollama","type":"local","status":"available","size":"16GB","context":131072},
            {"name":"phi-3.5-mini","provider":"ollama","type":"local","status":"available","size":"2.2GB","context":131072},
        ],
        "count":4,
    })

@app.route("/api/providers")
def api_providers():
    return jsonify({
        "providers":[
            {"name":"ollama","type":"local","status":"available","auth_methods":["none"]},
            {"name":"lm-studio","type":"local","status":"available","auth_methods":["none"]},
            {"name":"openai","type":"cloud","status":"configurable","auth_methods":["api_key"]},
            {"name":"openrouter","type":"cloud","status":"configurable","auth_methods":["api_key"]},
            {"name":"anthropic","type":"cloud","status":"configurable","auth_methods":["api_key","oauth"]},
        ],
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, debug=True)
