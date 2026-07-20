from contract_support import run_contract
from bounded_producer_consumer import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
