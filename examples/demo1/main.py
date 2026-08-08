import os
import streamlit as st
from dotenv import load_dotenv
from google.genai import types
from google.adk.agents import LlmAgent
from google.adk.runners import InMemoryRunner
from google.adk.tools.mcp_tool.mcp_toolset import McpToolset
from google.adk.tools.mcp_tool.mcp_session_manager import StreamableHTTPConnectionParams

load_dotenv()
st.set_page_config(page_title="ADK Client Setup")
st.title("Gemini + MCP Dashboard")

mcp_tools = McpToolset(
    connection_params=StreamableHTTPConnectionParams(
        url="http://127.0.0.1:8080/mcp",
    )
)

if "runner" not in st.session_state:
    agent = LlmAgent(
        name="Assistance",
        model="gemini-3.5-flash",
        instruction="""you are an operation assistant, use mcp server for live data.""",
        tools=[mcp_tools],
    )
    runner = InMemoryRunner(agent=agent, app_name="mcp_dashboard")
    runner.session_service.create_session_sync(
        app_name="mcp_dashboard",
        user_id="local_user",
        session_id="local_session",
    )
    st.session_state.runner = runner

if prompt := st.chat_input("Query your database..."):
    with st.chat_message("user"):
        st.markdown(prompt)
    with st.chat_message("assistant"):
        with st.spinner("Processing tools via MCP..."):
            final_text = ""
            for event in st.session_state.runner.run(
                user_id="local_user",
                session_id="local_session",
                new_message=types.Content(role="user", parts=[types.Part(text=prompt)]),
            ):
                if event.content and event.content.parts:
                    for part in event.content.parts:
                        if part.text:
                            final_text += part.text
            st.markdown(final_text)
