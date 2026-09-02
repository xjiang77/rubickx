from contract_support import run_contract
from transactional_outbox import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
