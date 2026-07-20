from contract_support import run_contract
from future_promise import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
