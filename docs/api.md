# API Reference

A detailed guide on the core components of the Parevo Flow API.

---

## ⚙️ Engine

`Engine` is the central component responsible for registering workflows, scheduling task claims, dispatching signals, and tracking execution states.

### Constructor
```go
engine := flow.NewEngine(storage, registry)
```

### Methods

#### `RegisterWorkflow`
Registers a workflow blueprint in the specified namespace.
```go
err := engine.RegisterWorkflow(ctx, "default", workflow)
```

#### `Execute`
Starts a new execution instance of a registered workflow. Accepts JSON payload input as `[]byte`.
```go
execID, err := engine.Execute(ctx, "default", "workflow-id", []byte(`{"param": "val"}`))
```

#### `GetExecution`
Retrieves the execution status and output details by ID.
```go
exec, err := engine.GetExecution(ctx, "default", execID)
```

#### `GetExecutionSteps`
Retrieves individual task status, outputs, or error details for an execution.
```go
steps, err := engine.GetExecutionSteps(ctx, "default", execID)
```

#### `CancelExecution`
Transitions a running execution to the `CANCELLED` state.
```go
err := engine.CancelExecution(ctx, "default", execID)
```

#### `SignalExecution`
Sends an external signal to resume a waiting step (`flow.NodeTypeSignal`) in a workflow.
```go
err := engine.SignalExecution(ctx, "default", execID, "node-id", `{"status":"approved"}`)
```

#### `StartWorker`
Launches the worker processing loop for the specified namespace.
```go
go engine.StartWorker(ctx, "default", "worker-id")
```

---

## 🗂️ Registry

`Registry` maintains the mapping of custom Go functions and custom `NodeExecutor` types.

### Methods

#### `RegisterFunction`
Registers a Go function as a reusable task step.
```go
registry.RegisterFunction("SendNotification", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // custom logic here
    return map[string]interface{}{"sent": true}, nil
})
```

#### `Register`
Registers a custom node executor implementing the `NodeExecutor` interface.
```go
registry.Register("custom_node_type", &MyCustomNodeExecutor{})
```

---

## 👷 Worker

`Worker` runs a background task polling loop to claim ready steps and execute them.

```go
worker := flow.NewWorker("worker-id", engine, registry, 100*time.Millisecond)
worker.SetNamespace("default")
worker.Start(ctx)
```

---

## 🎨 Workflow Builder

`WorkflowBuilder` provides a fluent API for building workflows.

### Methods

#### `NewWorkflow`
Creates a new workflow builder instance.
```go
builder := flow.NewWorkflow("workflow-id", "Workflow Name")
```

#### `AddNode`
Adds a step to the workflow and returns a `NodeBuilder`.
```go
node := builder.AddNode("node-id", flow.NodeTypeFunction)
```

#### `WithConfig`
Injects configuration values into a node.
```go
node.WithConfig("key", "value")
```

#### `WithRetry`
Defines retry attempts for a node.
```go
node.WithRetry(3)
```

#### `WithSaga`
Defines a compensation node to run on failure (Saga Pattern).
```go
node.WithSaga("compensation-node-id")
```

#### `Then`
Connects the current node to a target node.
```go
node.Then("next-node-id")
```

#### `If`
Connects the current node to a target node conditionally (branch routing).
```go
node.If("yes-node-id", "true")
```

#### `Connect`
Connects two nodes directly.
```go
builder.Connect("source-id", "target-id")
```

#### `ConnectIf`
Connects two nodes conditionally.
```go
builder.ConnectIf("source-id", "target-id", "condition")
```

#### `Build`
Compiles and returns the finalized `*flow.Workflow`.
```go
wf := builder.Build()
```
