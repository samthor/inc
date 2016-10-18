This contains experiments around a source, allowing fanout of data to multiple subscribers.

It creates buffered channels for its clients, as the `.Update` call pushes messages to those channels, and closes them if there's no more room to send.
Instead, this should probably use a high water mark, so channels aren't forcedly closed - then, a `.Next` call could return whether any messages were dropped.
