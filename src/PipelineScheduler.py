"""
Simple pure-Python cron parser and scheduler for Mimir-AIP pipeline runner.
Supports standard 5-field cron syntax: minute hour day month weekday.
Handles *, lists, ranges, and steps. No external dependencies.

Design notes:
- No support for named months/days, special strings, DST, or seconds field.
- Robust and easily extensible for future needs.
- Provides: CronSchedule class with is_now and next_run methods.
"""
import datetime
import time
from typing import List, Optional

class CronSchedule:
    def __init__(self, expr: str):
        self.expr = expr
        self.fields = self.parse(expr)

    @staticmethod
    def parse(expr: str) -> List[List[int]]:
        # Parse a standard 5-field cron expression into lists of valid values for each field
        # Fields: minute hour day month weekday
        def expand(field, minval, maxval):
            vals = set()
            for part in field.split(','):
                if part == '*':
                    vals.update(range(minval, maxval + 1))
                elif '/' in part:
                    base, step = part.split('/')
                    step = int(step)
                    if base == '*':
                        base_range = range(minval, maxval + 1)
                    elif '-' in base:
                        start, end = map(int, base.split('-'))
                        base_range = range(start, end + 1)
                    else:
                        base_range = [int(base)]
                    vals.update(v for v in base_range if (v - minval) % step == 0)
                elif '-' in part:
                    start, end = map(int, part.split('-'))
                    vals.update(range(start, end + 1))
                else:
                    vals.add(int(part))
            return sorted(vals)
        fields = expr.strip().split()
        if len(fields) != 5:
            raise ValueError(f"Invalid cron expression: {expr}")
        minute = expand(fields[0], 0, 59)
        hour = expand(fields[1], 0, 23)
        day = expand(fields[2], 1, 31)
        month = expand(fields[3], 1, 12)
        weekday = expand(fields[4], 0, 6)  # 0 = Sunday
        return [minute, hour, day, month, weekday]

    def is_now(self, dt: Optional[datetime.datetime] = None) -> bool:
        if dt is None:
            dt = datetime.datetime.now()
        minute, hour, day, month, weekday = self.fields
        return (
            dt.minute in minute and
            dt.hour in hour and
            dt.day in day and
            dt.month in month and
            dt.weekday() in weekday
        )

    def next_run(self, after: Optional[datetime.datetime] = None) -> datetime.datetime:
        # Returns the next datetime after 'after' that matches the cron schedule
        if after is None:
            after = datetime.datetime.now().replace(second=0, microsecond=0) + datetime.timedelta(minutes=1)
        else:
            after = after.replace(second=0, microsecond=0) + datetime.timedelta(minutes=1)
        # Brute-force search up to 366 days ahead (should be fast for most cron jobs)
        for _ in range(0, 366*24*60):
            if self.is_now(after):
                return after
            after += datetime.timedelta(minutes=1)
        raise RuntimeError("No matching time found in next 366 days for cron schedule: " + self.expr)

# Example usage (for test/demo):
if __name__ == "__main__":
    cron = CronSchedule("*/15 9-17 * * 1-5")  # Every 15 min during work hours, Mon-Fri
    now = datetime.datetime.now()
    print(f"Now: {now}")
    print(f"Matches now? {cron.is_now(now)}")
    print(f"Next run: {cron.next_run(now)}")
