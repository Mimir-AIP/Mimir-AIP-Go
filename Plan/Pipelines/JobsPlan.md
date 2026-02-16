# Jobs

Jobs are simply a way to setup the repeated/recurring execution of a pipeline or multiple pipelines. When a job is created, the user specifies which pipeline(s) to run and the schedule using a cron format. Mimir will then automatically trigger the specified pipeline(s) according to the schedule. 

## Job format
Similar to pipelines, jobs will also be defined in a YAML format. The job definition will include the name of the job, the pipeline(s) to run, and the schedule. The user will either be able to enter the raw yaml or use a form in the UI which will then generate the yaml for them.

## Job YAML Schema:
- **name**: string (required) - The unique name of the job.
- **pipelines**: array (required) - A list of pipeline names to execute according to the schedule.
- **schedule**: string (required) - The cron expression defining the schedule for the job.

## Job Example
This example demonstrates a job that runs two pipelines, `daily_data_ingestion` and `daily_data_processing`, every day at midnight.

```yaml
name: daily_data_job
pipelines:
    - daily_data_ingestion
    - daily_data_processing
schedule: "0 0 * * *"
```
