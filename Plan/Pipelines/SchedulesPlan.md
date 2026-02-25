# Schedules

Schedules are simply a way to setup the repeated/recurring execution of a pipeline or multiple pipelines. When a schedule is created, the user specifies which pipeline(s) to run and the schedule using a cron format. Mimir will then automatically trigger the specified pipeline(s) according to the schedule. 

## Schedule format
Similar to pipelines, schedules will also be defined in a YAML format. The schedule definition will include the name of the schedule, the pipeline(s) to run, and the cron schedule. The user will either be able to enter the raw yaml or use a form in the UI which will then generate the yaml for them.

## Schedule YAML Schema:
- **name**: string (required) - The unique name of the schedule.
- **pipelines**: array (required) - A list of pipeline names to execute according to the cron schedule.
- **cron_schedule**: string (required) - The cron expression defining when the schedule runs.

## Schedule Example
This example demonstrates a schedule that runs two pipelines, `daily_data_ingestion` and `daily_data_processing`, every day at midnight.

```yaml
name: daily_data_schedule
pipelines:
    - daily_data_ingestion
    - daily_data_processing
cron_schedule: "0 0 * * *"
```
