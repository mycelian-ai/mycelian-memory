"""PromptBuilder class for constructing system prompts."""

from typing import Dict, Any
import os
import asyncio
import logging


class PromptBuilder:
    """Handles prompt construction and template management."""
    
    # Constants
    MEMORY_TYPE = "chat"
    RULES_FILE = "../../prompts/context_summary_rules.md"
    FLUSH_INTERVAL = 6
    
    def __init__(self, vault_id: str, memory_id: str, mcp_client: Any, server_name: str):
        """Initialize PromptBuilder with vault and memory IDs."""
        self.vault_id = vault_id
        self.memory_id = memory_id
        self.mcp_client = mcp_client
        self.server_name = server_name
        
        # Load prompts on initialization
        self._load_prompts()
    
    def _load_prompts(self) -> None:
        """Load prompt templates from files and MCP."""
        # Read local rules file
        local_rules_path = os.path.join(os.path.dirname(__file__), self.RULES_FILE)
        try:
            with open(local_rules_path, "r", encoding="utf-8") as f:
                self.rules = f.read()
        except FileNotFoundError:
            self.rules = ""
        except IOError as e:
            logging.warning(f"Failed to read rules file: {e}")
            self.rules = ""

        # Get prompts from MCP using the correct pattern
        prompts = self._call_mcp_tool("get_default_prompts", {"memory_type": self.MEMORY_TYPE})
        
        # Handle both dict and string responses
        if isinstance(prompts, dict):
            templates = prompts.get("templates") or {}
            self.entry_capture_prompt = templates.get("entry_capture_prompt") or ""
            self.summary_prompt = templates.get("summary_prompt") or ""
            self.context_prompt = templates.get("context_prompt") or ""
        else:
            # If we get a string or other type, use empty prompts
            self.entry_capture_prompt = ""
            self.summary_prompt = ""
            self.context_prompt = ""
    
    def _call_mcp_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Call an MCP tool by name using the correct pattern.
        
        Args:
            tool_name: Name of the tool to call
            arguments: Arguments to pass to the tool
            
        Returns:
            Tool response or empty dict on failure
        """
        try:
            # Get all tools
            async def _get_tools():
                return await self.mcp_client.get_tools()  # type: ignore[attr-defined]
            
            tools = asyncio.run(_get_tools())
            
            # Find the specific tool
            tool = None
            for t in tools:
                if getattr(t, "name", None) == tool_name:
                    tool = t
                    break
            
            if tool is None:
                logging.warning(f"MCP tool not found: {tool_name}")
                return {}
            
            # Invoke the tool
            if hasattr(tool, "ainvoke"):
                result = asyncio.run(tool.ainvoke(arguments))  # type: ignore[attr-defined]
            elif hasattr(tool, "invoke"):
                result = tool.invoke(arguments)  # type: ignore[attr-defined]
            else:
                logging.warning(f"Tool {tool_name} has no invoke method")
                return {}
            
            return result or {}
            
        except Exception as e:
            logging.warning(f"Failed to call MCP tool {tool_name}: {e}")
            return {}
    
    def build_system_prompt(self) -> str:
        """Construct the complete system prompt with all rules and templates."""
        return (
            # IDENTITY & ROLE
            "You are the Mycelian Memory Agent. You OBSERVE conversations between USER and AI ASSISTANT "
            "without role-playing either participant. Your task: capture durable memory using MCP tools.\n\n"
            
            # TOOL PARAMETERS
            f"VAULT_ID: '{self.vault_id}'\n"
            f"MEMORY_ID: '{self.memory_id}'\n"
            "Use these IDs for all MCP tool calls requiring vault_id/vaultId or memory_id/memoryId.\n\n"
            
            # MESSAGE TYPES IN YOUR CONTEXT
            "MESSAGE TYPES YOU WILL SEE:\n"
            "• SystemMessage with message_type='prompt': Instructions for you (not part of conversation)\n"
            "• SystemMessage with message_type='control': Commands to execute (SESSION_START, SESSION_END)\n"
            "• ChatMessage with 'idx' field: Conversation messages to persist (check the idx value!)\n"
            "• AIMessage with tool_calls: Your previous tool invocations\n"
            "• ToolMessage: Results from tools (get_context, list_entries, etc.)\n\n"
            
            # CONTROL COMMAND ACTIONS
            "CONTROL COMMAND ACTIONS:\n"
            "• SESSION_START → Call get_context(), then list_entries(limit=10)\n"
            "• SESSION_END → No action required\n\n"
            
            # CONVERSATION MESSAGE ACTIONS
            "CONVERSATION MESSAGE ACTIONS:\n"
            "• Call add_entry ONLY for NEW ChatMessages that haven't been processed\n"
            "• A ChatMessage is processed if there's already an add_entry call for it\n"
            "• Check the 'idx' field on the current ChatMessage\n"
            f"• When idx % {self.FLUSH_INTERVAL} == 0: After add_entry, call await_consistency() then put_context()\n\n"
            
            # STRICT RULES
            "STRICT RULES:\n"
            "• ONE add_entry per conversation message (no duplicates)\n"
            "• Tags: {{\"role\": \"user\"}} or {{\"role\": \"assistant\"}} only\n"
            "• NEVER persist control messages (SESSION_START, SESSION_END)\n"
            "• CRITICAL: Look at AIMessage tool calls to see what's been done already\n"
            "• If you see AIMessage with add_entry for a message's content, it's ALREADY PROCESSED\n"
            "• Use tool results (get_context, list_entries) as context for summaries\n"
            "• Emit ONLY MCP tool calls (no explanatory text)\n"
            "• IMPORTANT: If there are no NEW messages to process, return empty (no tool calls)\n"
            "• When the last ToolMessage is 'enqueued' and no new ChatMessage follows, STOP\n\n"
            
            # Append the dynamic prompts and rules
            + str(self.rules).strip() + "\n\n"
            + str(self.entry_capture_prompt).strip() + "\n\n" 
            + str(self.summary_prompt).strip() + "\n\n"
            + str(self.context_prompt).strip()
        ).strip()