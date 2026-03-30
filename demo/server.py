"""
BuzzBench demo server — simple FastAPI app for local load testing.

Endpoints:
  GET  /health        Always fast, always 200. Good for baseline tests.
  GET  /users/{id}    Simulates a DB lookup with a small random delay.
  POST /orders        Accepts a JSON body, returns a created order.
  GET  /slow          Takes 200–500 ms. Shows spread in min/max/avg.
  GET  /flaky         Returns 500 ~30% of the time. Demonstrates error rate reporting.

Run with:
  uvicorn demo.server:app --host 127.0.0.1 --port 8000
"""

import random
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI(title="BuzzBench Demo Server")


@app.get("/health")
def health():
    return {"status": "ok", "timestamp": time.time()}


@app.get("/users/{user_id}")
def get_user(user_id: int):
    if user_id < 1:
        raise HTTPException(status_code=404, detail="User not found")
    # Simulate a small DB read delay
    time.sleep(random.uniform(0.01, 0.05))
    return {
        "id": user_id,
        "name": f"User {user_id}",
        "email": f"user{user_id}@example.com",
    }


class OrderRequest(BaseModel):
    product_id: int
    quantity: int = 1


@app.post("/orders", status_code=201)
def create_order(order: OrderRequest):
    # Simulate a write delay slightly longer than a read
    time.sleep(random.uniform(0.02, 0.08))
    return {
        "order_id": random.randint(10000, 99999),
        "product_id": order.product_id,
        "quantity": order.quantity,
        "status": "created",
    }


@app.get("/slow")
def slow():
    delay = random.uniform(0.2, 0.5)
    time.sleep(delay)
    return {"message": "slow response", "delay_ms": round(delay * 1000)}


@app.get("/flaky")
def flaky():
    if random.random() < 0.3:
        raise HTTPException(status_code=500, detail="Random failure")
    return {"message": "ok"}
