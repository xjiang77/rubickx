from threading import Barrier, Event, Lock, Thread

from contract_support import run_contract
from structured_concurrency_cancellation import evaluate


def test_shared_contract():
    run_contract(__file__, evaluate)


def test_failure_cancels_sibling_and_all_threads_join():
    start = Barrier(3)
    cancelled = Event()
    lock = Lock()
    events = []

    def failing_child():
        start.wait()
        with lock:
            events.append("failed")
        cancelled.set()

    def sibling():
        start.wait()
        cancelled.wait()
        with lock:
            events.append("cancelled")

    threads = [Thread(target=failing_child), Thread(target=sibling)]
    for thread in threads:
        thread.start()
    start.wait()
    for thread in threads:
        thread.join()

    assert events == ["failed", "cancelled"]
    assert all(not thread.is_alive() for thread in threads)
