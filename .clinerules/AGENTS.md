Follow instructions in [DevelopmentPlan.md](DevelopmentPlan.md) to complete the assigned task.
Stack: Backend- Go, Frontent - Static HTML/CSS/JS
For deployment: Kubernetes(local dev with rancher desktop)

Testing Strategy:
- Unit tests for individual components(can be executed with `go test` for Go code) THESE CANNOT BE SKIPPED OR MOCKED, YOU MUST WRITE REAL TESTS FOR EACH COMPONENT AS YOU BUILD IT
- Integration tests should be testing against a running instance of the system(system running in kubernetes) against the actual API endpoints THESE CANNOT BE SKIPPED OR MOCKED, YOU MUST WRITE REAL INTEGRATION TESTS THAT INTERACT WITH THE ACTUAL DEPLOYED SYSTEM

Development strategy:
- Follow the development plan step by step, do not skip any steps or mock any components.
- Work in small increments, completing one step at a time and testing thoroughly before moving on to the next step.(previous step must be fully complete and tests must pass without skipping or mocking before moving on to the next step)