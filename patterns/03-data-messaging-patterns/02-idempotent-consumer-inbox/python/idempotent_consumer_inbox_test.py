from contract_support import run_contract
from idempotent_consumer_inbox import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
