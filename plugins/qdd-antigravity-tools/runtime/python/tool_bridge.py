"""
QDD Tool Bridge for Antigravity
Exposes governed QDD tools and handles validation.
Strictly follows Zero-Else and Early Return.
"""

from typing import Dict, Any, Callable, Optional

class QDDToolBridge:
    def __init__(self):
        self._tools: Dict[str, Dict[str, Any]] = {}

    def register_tool(
        self,
        name: str,
        description: str,
        parameters_schema: Dict[str, Any],
        handler: Callable[..., Any]
    ) -> None:
        if not name or not callable(handler):
            raise ValueError("Tool must have a valid name and callable handler")

        self._tools[name] = {
            "name": name,
            "description": description,
            "parameters": parameters_schema,
            "handler": handler
        }

    def list_tools(self) -> list:
        return [
            {
                "name": t["name"],
                "description": t["description"],
                "parameters": t["parameters"]
            }
            for t in self._tools.values()
        ]

    async def execute_tool(self, name: str, arguments: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        args = arguments or {}
        tool = self._tools.get(name)

        if not tool:
            return {
                "status": "ERROR",
                "error": f"Tool '{name}' not found in QDDToolBridge"
            }

        try:
            handler = tool["handler"]
            import inspect
            if inspect.iscoroutinefunction(handler):
                result = await handler(**args)
                return {"status": "SUCCESS", "result": result}

            result = handler(**args)
            return {"status": "SUCCESS", "result": result}
        except Exception as e:
            return {"status": "ERROR", "error": str(e)}
