# Built-in Node Types

Parevo Flow comes pre-registered with several standard task node types.

---

## 🧩 Node Specifications

### 1. `function`
Runs custom Go functions registered in the handler registry.
*   **Config Options**:
    *   `function` (string, required): The registered function name.
*   **Example**:
    ```json
    {
      "id": "fetch_user",
      "type": "function",
      "config": {
        "function": "FetchUserFromDB"
      }
    }
    ```

### 2. `http`
Makes external HTTP requests.
*   **Config Options**:
    *   `url` (string, required): Destination endpoint.
    *   `method` (string, optional): HTTP method (`GET`, `POST`, etc. Defaults to `GET`).
    *   `headers` (map, optional): Custom headers.
*   **Example**:
    ```json
    {
      "id": "fetch_status",
      "type": "http",
      "config": {
        "url": "https://api.example.com/v1/status",
        "method": "GET",
        "headers": {
          "Accept": "application/json"
        }
      }
    }
    ```

### 3. `condition`
Evaluates a comparison expression and routes downstream nodes to either `true` or `false` branches.
*   **Config Options**:
    *   `variable` (string, required): Input JSON key to read.
    *   `operator` (string, required): `==`, `!=`, `>`, `<`, `>=`, `<=`, `contains`, `not_contains`.
    *   `value` (any, required): Value to compare against.
*   **Example**:
    ```json
    {
      "id": "check_age",
      "type": "condition",
      "config": {
        "variable": "age",
        "operator": ">=",
        "value": 18
      }
    }
    ```

### 4. `switch`
Multi-way branching based on input field values.
*   **Config Options**:
    *   `variable` (string, required): Input JSON key to read.
    *   `cases` (map, required): Value-to-branch-label mappings.
    *   `default` (string, optional): Default branch if no case matches (defaults to `default`).
*   **Example**:
    ```json
    {
      "id": "route_role",
      "type": "switch",
      "config": {
        "variable": "role",
        "cases": {
          "admin": "admin-branch",
          "user": "user-branch"
        },
        "default": "guest-branch"
      }
    }
    ```

### 5. `signal`
Suspends execution path and waits for an external REST API signal to resume.
*   **Config Options**:
    *   `timeout` (string, optional): Duration format (e.g., `24h`, `7d`). If exceeded, the task fails.
*   **Example**:
    ```json
    {
      "id": "wait_approval",
      "type": "signal",
      "config": {
        "timeout": "48h"
      }
    }
    ```

### 6. `subworkflow`
Runs a child workflow nested inside the parent workflow, polling for its completion.
*   **Config Options**:
    *   `workflowId` (string, required): ID of the child workflow to trigger.
    *   `namespace` (string, optional): Namespace of the child workflow.
*   **Example**:
    ```json
    {
      "id": "run_onboarding",
      "type": "subworkflow",
      "config": {
        "workflowId": "employee-onboarding",
        "namespace": "default"
      }
    }
    ```

### 7. `ai`
Sends prompt requests to OpenAI, Anthropic, or Google Gemini.
*   **Config Options**:
    *   `provider` (string, required): `openai`, `anthropic`, or `gemini`.
    *   `api_key` (string, required): API authorization key.
    *   `model` (string, required): Model name (e.g., `gpt-4o`, `claude-3-5-sonnet`, `gemini-1.5-pro`).
    *   `prompt` (string, required): Text template supporting `{{.field}}` template syntax.
    *   `system_prompt` (string, optional): System instructions.
    *   `temperature` (float, optional): Randomness value (0.0-2.0. Defaults to `0.7`).
    *   `max_tokens` (int, optional): Output token limit (defaults to `1000`).
    *   `result_key` (string, optional): Target JSON key to store response (defaults to `ai_response`).
*   **Example**:
    ```json
    {
      "id": "ai_summarize",
      "type": "ai",
      "config": {
        "provider": "openai",
        "api_key": "YOUR_OPENAI_KEY",
        "model": "gpt-4o",
        "prompt": "Summarize this customer feedback: {{.feedback}}",
        "result_key": "summary"
      }
    }
    ```

### 8. `notify`
Sends webhook payloads to external URLs using templated details.
*   **Config Options**:
    *   `url` (string, required): Destination URL (can contain `{{.field}}` parameters).
    *   `method` (string, optional): Defaults to `POST`.
    *   `body` (string, optional): Go template for webhook body. Forewards input if empty.
*   **Example**:
    ```json
    {
      "id": "slack_notify",
      "type": "notify",
      "config": {
        "url": "https://hooks.slack.com/services/T00/B00/X00",
        "method": "POST",
        "body": "{\"text\": \"New sign up: {{.user.name}} ({{.user.email}})\"}"
      }
    }
    ```

### 9. `transform`
Reshapes and filters JSON context objects using Go templates.
*   **Config Options**:
    *   `mapping` (map, required): Key-to-template pairs.
*   **Example**:
    ```json
    {
      "id": "format_payload",
      "type": "transform",
      "config": {
        "mapping": {
          "fullName": "{{.firstName}} {{.lastName}}",
          "contactEmail": "{{.email}}"
        }
      }
    }
    ```

### 10. `wait`
Inserts time delays into the workflow.
*   **Config Options**:
    *   `duration` (string, required): Duration string format (e.g., `5s`, `10m`, `2h`).
*   **Example**:
    ```json
    {
      "id": "wait_5_sec",
      "type": "wait",
      "config": {
        "duration": "5s"
      }
    }
    ```

### 11. `setvariable`
Overwrites or injects variables into the execution JSON payload.
*   **Config Options**:
    *   `variables` (map, required): Map of variable key/value pairs.
*   **Example**:
    ```json
    {
      "id": "set_meta",
      "type": "setvariable",
      "config": {
        "variables": {
          "status": "pending_billing",
          "processed_by": "engine-v1"
        }
      }
    }
    ```

### 12. `log`
Logs messages to the console for tracking.
*   **Config Options**:
    *   `message` (string, required): Output text format.
*   **Example**:
    ```json
    {
      "id": "audit_log",
      "type": "log",
      "config": {
        "message": "Step reached successfully!"
      }
    }
    ```
