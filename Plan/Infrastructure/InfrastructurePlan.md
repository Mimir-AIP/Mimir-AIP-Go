# Infrastructure

### Frontend single kubernetes container
### Backend Orchestrater single kubernetes container
### Backend workers(can be used to run pipelines, ml models(inference & training), digital twin jobs etc.) scalable kubernetes workers(one worker per job; orchestrater spins this up, worker completes job, returns results to orchestrator and closes)
^Initially during dev this will all be on single system but in theory the workers should be able to scale across a cluster of systems or even remote systems for distributed sclaing.