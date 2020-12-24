# Slack Queue App

A Slack bot to maintain a set of per-channel queues (e.g., for tutoring, help desk, etc.).

## About

The bot supports four commands/actions:

* Creating or deleting a per-channel queue.
  * This operation is restricted to global administrators.
  * Channel queues can be created with an optional admin channel that will
    receive notifications about queue state.
  * If a queue has an admin channel, only users in that channel may dequeue or
    remove users from the queue.
* In a channel, users can enqueue themselves via a slash (`/`) command. Any text
  after the command is stored as metadata.
* The queue state can be listed by admins using a list slash command.
  * In the UI response, users can be dequeued, removed, or moved up/down the
    queue.
* Admins may also dequeue the first user in the queue via another slash command.

If an optional persistence flag is supplied, application state and queue state
is persisted across restarts.

### License

This module is licensed under the [Mozilla Public License, version
2.0](https://www.mozilla.org/en-US/MPL/).
