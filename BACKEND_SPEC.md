MomLaunchpad Backend — MVP Technical Specification

1. Purpose

This document defines the authoritative backend architecture for the MomLaunchpad MVP.

The backend is responsible for:

Conversational intelligence

Memory & logging

Calendar intelligence

Language handling

Cost control

Security

The backend is the single source of truth.

2. MVP Feature Scope
✅ Included

Conversational Q&A (text-first, audio supported indirectly)

Rule-based intent classification

AI-assisted pregnancy support

Intelligent calendar suggestions

Multilingual handling

Admin language management

⚠️ Optional (Probe Only)

Savings (manual, non-financial, no automation)
Purely informational, can be disabled entirely.

❌ Explicitly Excluded

Payments

Automated deductions

Product recommendations

Doctor dashboards

Analytics pipelines

Vector search (Qdrant)

Tiny LLMs

Microservices

3. Tech Stack
Layer	Choice
Language	Go
Framework	Gin or Fiber
Transport	HTTP + WebSocket
Primary DB	PostgreSQL
Cache	Redis (optional)
AI	DeepSeek API
Auth	JWT
Hosting	Single VM
Testing	Go test (table-driven, TDD)
4. High-Level Architecture
Client (Flutter)
  ├─ HTTP → Admin / Calendar / Auth
  └─ WebSocket → Chat (streaming)

Backend (Go)
  ├─ Rule-based classifier
  ├─ Memory manager
  ├─ Prompt builder
  ├─ DeepSeek client
  ├─ Calendar engine
  └─ Language manager

5. Transport Rules
WebSocket (Chat only)

/ws/chat

JWT validated on connection

Used for:

User messages

Streaming AI responses

Stateless protocol, session-aware application logic

HTTP

Auth

Calendar CRUD

Admin APIs

Language management

6. Intent Classification (Critical)
Rule-Based Classifier (First Gate)

All incoming chat messages pass through a deterministic classifier.

Intent Categories

- small_talk
- pregnancy_question
- symptom_report
- scheduling_related
- unclear


Classifier Inputs

Normalized text

Language code

Classifier Outputs

{
  "intent": "pregnancy_question",
  "confidence": 0.82
}


No LLM is used here.

7. Small Talk Handling
Small Talk Rules

Never triggers memory loading

Never builds a super-prompt

Never writes to DB

Uses canned or templated responses

Example responses:

“I’m here with you. How can I help today?”

“Thanks for checking in. What’s on your mind?”

Small talk is treated as a UX concern, not an AI concern.

8. Pregnancy-Related Flow (Super-Prompt Path)

Triggered only for:

pregnancy_question

symptom_report

Backend Pipeline
Incoming message
↓
Intent classifier
↓
Load short-term memory (last 5–10 messages)
↓
Load long-term facts (pregnancy week, diet, etc.)
↓
Build language-aware super-prompt
↓
Call DeepSeek
↓
Stream response via WebSocket
↓
Extract new facts (if any)
↓
Suggest calendar action (if applicable)

9. Super-Prompt Rules

The super-prompt may include:

Pregnancy stage (if known)

Relevant long-term facts

Recent messages only if relevant

Language-specific system instructions

The super-prompt must never include:

Small talk

UX fillers

Irrelevant history

Raw admin configuration

10. Memory Model
Short-Term Memory

Last N messages (5–10)

Conversation continuity only

Not permanent

Long-Term Memory

Extracted facts only

Key/value with confidence score

Example:

pregnancy_week = 14 (0.9)
diet = vegetarian (0.7)


Memory extraction is AI-assisted but backend-approved.

11. Database (Postgres)
Core Tables

users

messages

user_facts

reminders

languages

savings_entries (optional, manual only)

No ORMs required.

12. Calendar Intelligence

The backend:

Detects scheduling relevance

Suggests (never creates) reminders

Example suggestion:

{
  "type": "calendar",
  "message": "Would you like to set a reminder to monitor this symptom?"
}


Reminder creation requires explicit user confirmation via HTTP API.

13. Multilingual Support
Rules

Backend never guesses language

Language code is trusted only after validation

Unsupported languages fallback to English

Admin Controls

Enable / disable languages

Mark languages as experimental

14. Security & Cost Controls

JWT middleware

Rate limiting on chat

Cache repeated queries cautiously

Never cache sensitive medical responses long-term

No LLM keys exposed

15. Testing (Mandatory)

Every domain must have:

Pure functions

Table-driven tests

Mocked DeepSeek client

Test categories:

Intent classification

Prompt construction

Memory extraction

Language fallback

Calendar suggestion logic

16. Design Principles

Backend is the brain

AI is a dependency

Determinism before intelligence

MVP discipline over cleverness
