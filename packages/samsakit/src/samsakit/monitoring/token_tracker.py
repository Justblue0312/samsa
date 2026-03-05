"""Token Tracker - Track token usage and costs."""

import time
from dataclasses import dataclass, field
from typing import Optional

try:
    import tiktoken
except ImportError:
    tiktoken = None

from ..config import settings

# Token pricing (USD per 1M tokens) - GPT-4o-mini
TOKEN_PRICING = {
    "input": 0.15,  # $0.15 per 1M input tokens
    "output": 0.60,  # $0.60 per 1M output tokens
}


@dataclass
class AgentUsage:
    """Usage statistics for an agent."""

    name: str
    total_tokens: int = 0
    input_tokens: int = 0
    output_tokens: int = 0
    elapsed_ms: int = 0
    requests: int = 0

    @property
    def cost_usd(self) -> float:
        """Calculate cost in USD."""
        input_cost = (self.input_tokens / 1_000_000) * TOKEN_PRICING["input"]
        output_cost = (self.output_tokens / 1_000_000) * TOKEN_PRICING["output"]
        return input_cost + output_cost


@dataclass
class TokenTracker:
    """Track token usage across agents."""

    usages: dict[str, AgentUsage] = field(default_factory=dict)
    start_time: float = field(default_factory=time.time)

    def add_usage(
        self,
        agent_name: str,
        usage,  # pydantic_ai RunUsage
        elapsed_ms: int = 0,
    ) -> None:
        """Add usage from an agent call."""
        if agent_name not in self.usages:
            self.usages[agent_name] = AgentUsage(name=agent_name)

        u = self.usages[agent_name]
        u.total_tokens += usage.total_tokens
        u.input_tokens += usage.input_tokens
        u.output_tokens += usage.output_tokens
        u.elapsed_ms += elapsed_ms
        u.requests += 1

    def estimate_tokens(self, text: str) -> int:
        """Estimate token count using tiktoken."""
        if tiktoken is None:
            # Rough estimate: ~4 chars per token
            return len(text) // 4

        try:
            encoding = tiktoken.encoding_for_model("gpt-4o-mini")
            return len(encoding.encode(text))
        except Exception:
            return len(text) // 4

    def get_summary(self) -> dict:
        """Get usage summary."""
        total_tokens = 0
        total_cost = 0.0
        total_elapsed = 0
        agents = []

        for usage in self.usages.values():
            total_tokens += usage.total_tokens
            total_cost += usage.cost_usd
            total_elapsed += usage.elapsed_ms

            agents.append(
                {
                    "name": usage.name,
                    "total_tokens": usage.total_tokens,
                    "elapsed_ms": usage.elapsed_ms,
                    "cost_usd": round(usage.cost_usd, 6),
                }
            )

        return {
            "agents": agents,
            "total_tokens": total_tokens,
            "total_cost_usd": round(total_cost, 6),
            "total_elapsed_ms": total_elapsed,
        }


class CostReporter:
    """Generate cost comparison reports."""

    @staticmethod
    def estimate_monthly_cost(
        requests_per_month: int,
        avg_tokens_per_request: int = 2500,
    ) -> dict:
        """Estimate monthly cost for AI agent approach."""
        # Based on GPT-4o-mini pricing
        input_tokens = int(avg_tokens_per_request * 0.6)
        output_tokens = int(avg_tokens_per_request * 0.4)

        input_cost = (input_tokens / 1_000_000) * TOKEN_PRICING["input"] * requests_per_month
        output_cost = (output_tokens / 1_000_000) * TOKEN_PRICING["output"] * requests_per_month

        return {
            "requests_per_month": requests_per_month,
            "avg_tokens_per_request": avg_tokens_per_request,
            "estimated_monthly_cost": round(input_cost + output_cost, 2),
            "cost_per_1k_requests": round((input_cost + output_cost) / (requests_per_month / 1000), 4),
        }

    @staticmethod
    def compare_with_traditional_ml(
        requests_per_month: int,
    ) -> dict:
        """Compare AI agent cost with traditional ML approach."""
        ai_cost = CostReporter.estimate_monthly_cost(requests_per_month)

        # Traditional ML estimates
        ml_inference_cost = requests_per_month * 0.000002  # $0.000002 per inference
        ml_retraining_monthly = 100  # Estimated $100/month for retraining
        ml_mlops_overhead = 50  # Estimated $50/month for MLOps

        return {
            "ai_agent": {
                "monthly_cost": ai_cost["estimated_monthly_cost"],
                "description": "AI agent with LLM",
            },
            "traditional_ml": {
                "monthly_cost": round(ml_inference_cost + ml_retraining_monthly + ml_mlops_overhead, 2),
                "description": "Inference + retraining + MLOps",
            },
            "recommendation": "AI agent" if ai_cost["estimated_monthly_cost"] < 200 else "Evaluate based on scale",
        }
