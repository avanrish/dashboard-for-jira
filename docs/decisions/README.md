# Architecture decision records

This directory records the decisions that shape the project: what was decided, why, and what else was considered.
The code shows what the project does; these records explain why it is that way.
They exist mainly for AI coding agents and new contributors, who see the code but not the reasoning behind it, and might otherwise undo deliberate choices or propose options that were already rejected.

## When to write one

Write a record only for a decision that shapes the whole project and is costly to reverse, such as the stack, how the app authenticates, where it keeps data, or how it is distributed.
Most changes need none.

## Format

- One file per decision, named `NNNN-short-title.md`, numbered one higher than the highest record here.
- Start from [`template.md`](template.md) and keep it short: a reader should finish in a couple of minutes.
- Describe the decision, not its implementation: leave out what the code already shows, so the record stays true as the code changes.
- Status is one of `Proposed`, `Accepted`, `Accepted, amended by NNNN`, `Rejected`, `Deprecated`, or `Superseded by NNNN`.

## Changing a decision

Once a record has been pushed, do not rewrite it; until then, edit it freely.
To replace a decision, write a new record, then set the old one's status to `Superseded by NNNN` and link the new one.
To change only part of a decision, write a new record that says what it changes, then set the old one's status to `Accepted, amended by NNNN` and link the new one.
Fixing typos or adding links is fine.
