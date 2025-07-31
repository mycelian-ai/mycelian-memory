---
title: Environment-Driven Configuration (12-Factor)
status: Accepted
date: 2025-07-21
---

# Context

Synapse services run in containers across local Docker, CI, and Cloud Run.  Static config files introduce state drift and secrets leakage; environment variables fit the 12-factor model and integrate with Secret Manager.

# Decision

1. All runtime configuration is provided via env-vars parsed with `kelseyhightower/envconfig`.  
2. Required keys: `SPANNER_PROJECT_ID`, `SPANNER_INSTANCE_ID`, `SPANNER_DATABASE_ID`, `PORT`, `ENV_STAGE`.  
3. Defaults exist only for local dev (e.g., `PORT=8080`); CI ensures required keys are set for other stages.

# Consequences

• Containers are stateless and portable across environments.  
• Secret injection (cloud-specific) is painless.  
• Changing config requires no image rebuild—only env-var tweaks in deployment YAML. 