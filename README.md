Can you help me figure out what's going on here? I'm going to rewrite it, but
I'd like to understand why it's not doing what I expect it to. You should be
able to run the test suite and get a failure

What the code is doing:

- Multiple different goroutines are taking a lock and then appending data to a buffer

- A "flush" goroutine calls sync.Cond.Wait() to wait for an incoming signal that data has been appended

- Each goroutine that appends to the buffer calls Signal() to try to wake up the "flush" goroutine

I expect that the flush goroutine will wake up after each call to Signal(),
check whether the batch is ready to be flushed, and if not go back to sleep.
What I see instead is that lots of other goroutines are taking out the lock
instead of the flush goroutine, and as a result we're dropping data.

I didn't expect that to happen based on my reading of the docs for sync.Cond,
which (to me) indicate that Signal() will wake up a goroutine that calls Wait().
Instead it looks like it's just unlocking any goroutine? Maybe this is because
the thread that is calling Signal() holds the lock?

Thanks very much for your help!
