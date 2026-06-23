# Hello World
Tags: hello_world

The simplest test: push one event, one webhook receives it.

## One Event One Delivery
* Create namespace "hello"
* Start target "myapp"
* Register event type "greeting"
* Register webhook "myapp" in current namespace subscribed to "greeting"
* Push event "greeting" with payload "{\"message\": \"Hello, World!\"}"
* Wait for "myapp" to receive "1" deliveries
* Target "myapp" should have received "1" deliveries
* Latest delivery to "myapp" has enveloped payload field "message" equal to "Hello, World!"
