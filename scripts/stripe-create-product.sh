
stripe products create \
  --name "Idea" \
  --description "StaticBackend idea plan"

stripe prices create \
  --product prod_Tpn2XOgjm4HdrP \
  --unit-amount 300 \
  --currency usd \
  --recurring.interval=month \
  --recurring.trial-period-days=14