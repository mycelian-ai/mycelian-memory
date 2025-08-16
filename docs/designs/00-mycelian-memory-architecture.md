# Mycelian Memory Architecture Doc

## Overview

This document introduces the primary architecture of Mycelian Memory and how it is used. It explains why a product like Mycelian is needed and frames the problem and current landscape the system addresses.

It then states the requirements the system currently meets, the tenets that guide design decisions and trade-offs, and the current design at a high level, including how extraction, storage, and retrieval work together. Use this as the entry point before diving into detailed designs.

## Problem

When ChatGPT launched in November 2022, it kicked off a rapid wave of model improvements and practical agents. These agents are LLM programs that use tools, plan multi-step workflows, and interact with APIs, files, and system interfaces. Today they write code, triage support tickets, teach students, coach users, and handle customer inquiries. New categories appear constantly. But most agents can't remember much beyond their current session, which limits how useful they can be over time.

Model providers keep pushing context windows larger. Some flagship models now handle 1M tokens as of August 2025. But even these expanded windows hit physical limits of model and machine memory. Real applications being built now will accumulate context for weeks, months, even years. Bigger is also not always better, many usecases will choose smaller, faster models for better latency and lower costs. Bigger windows are helpful, but they don't solve the fundamental need for persistent memory that lives outside the model.

Memory needs to be secure, durable, and above all factually correct. It will be needed for the smallest use cases from a single developer working with a brainstorming agent to a social media app creating memories for tens of millions of users with tens of thousands of entries per user. A generalized memory system must handle these constraints without compromising on the fundamentals. 

Current solutions include Graph RAG, which can deliver high‑quality, multi‑hop retrieval, but large graphs are hard to keep healthy. Schemas evolve, upserts and merges get tricky, edges go stale, and re‑indexing costs rise. All‑in‑one agent environments often bundle orchestration, tools, evaluators, and UIs, which dilutes focus on memory semantics, data governance, and observability.

The AI field thrives on openness. A memory platform needs to be inspectable and configurable. Developers should be able to quickly download and prototype. It should use a language that AI coding agents work well with and has a healthy ecosystem that's simple to learn and build with. Above all, this openness is needed for trust. Because no single approach fits every case, modularity and choice are essential. Teams should be able to swap embeddings, indexes, and storage backends without rewriting application logic, and reason about behavior through a clear, stable API.

Performance needs also vary. Critical paths demand top‑tier quality and low latency. Many everyday tasks accept average performance at lower cost. The right goal is flexibility: let users dial for cost, speed, and relevance instead of chasing a single “best” benchmark. 

## Current Landscape

## Assumptions

## Requirements

### Functional Requirements

### Non-functional Requirements

## Out-of-Scope

## Tenets

## High Level Design

## Hypothesis

## Extraction

### Option 1 - Client Generated Memory

### Option 2 - Encapsulated Memory

### Option 3 (Recommended) - Hybrid (Symbiotic Memory)

## Storage

## Memory as a log of immutable chronological events

## Context as a log of immutable Context Fragments

## Retrieval



