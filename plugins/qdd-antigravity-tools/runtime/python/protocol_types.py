"""
QDD Antigravity Protocol Types
Strictly follows Zero-Else and Early Return.
"""

from dataclasses import dataclass, field
from typing import Optional, Dict, Any, List
import time
import uuid

@dataclass
class PingRequest:
    prompt: str
    session_id: str = field(default_factory=lambda: f"sess_{uuid.uuid4().hex[:12]}")
    turn_id: int = 1
    protocol: str = "qdd-ping-pong/v1"
    message_type: str = "PING"
    context: Dict[str, Any] = field(default_factory=dict)
    options: Dict[str, Any] = field(default_factory=dict)
    timestamp: float = field(default_factory=time.time)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "protocol": self.protocol,
            "message_type": self.message_type,
            "session_id": self.session_id,
            "turn_id": self.turn_id,
            "timestamp": self.timestamp,
            "prompt": self.prompt,
            "context": self.context,
            "options": self.options,
        }

@dataclass
class PongResponse:
    session_id: str
    turn_id: int
    status: str
    content: str
    duration_ms: float
    protocol: str = "qdd-ping-pong/v1"
    message_type: str = "PONG"
    tools_executed: List[Dict[str, Any]] = field(default_factory=list)
    artifacts: List[str] = field(default_factory=list)
    error: Optional[str] = None
    timestamp: float = field(default_factory=time.time)

    def is_success(self) -> bool:
        return self.status == "SUCCESS"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "protocol": self.protocol,
            "message_type": self.message_type,
            "session_id": self.session_id,
            "turn_id": self.turn_id,
            "timestamp": self.timestamp,
            "status": self.status,
            "duration_ms": self.duration_ms,
            "content": self.content,
            "tools_executed": self.tools_executed,
            "artifacts": self.artifacts,
            "error": self.error,
        }
