from contract_support import run_contract
from publisher_subscriber import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
