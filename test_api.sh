#!/bin/bash

echo "🧪 COMPREHENSIVE API TEST SUITE"
echo "================================"

BASE_URL="http://localhost:3000"

echo ""
echo "✅ Test 1: Health Check"
echo "------------------------"
curl -s $BASE_URL/ | head -c 100
echo ""

echo ""
echo "✅ Test 2: User Registration"
echo "----------------------------"
TIMESTAMP=$(date +%s)
EMAIL="testuser$TIMESTAMP@example.com"
MEMBER_ID="LBK$TIMESTAMP"

REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"password123\",\"first_name\":\"Test\",\"last_name\":\"User\",\"phone\":\"081-111-1111\",\"birthday\":\"1990-01-01\",\"member_id\":\"$MEMBER_ID\"}" \
  $BASE_URL/register)

echo "Registration: $REGISTER_RESPONSE"

echo ""
echo "✅ Test 3: User Login"
echo "---------------------"
LOGIN_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"password123\"}" \
  $BASE_URL/login)

echo "Login: $LOGIN_RESPONSE"

# Extract token from login response
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: ${TOKEN:0:50}..."

echo ""
echo "✅ Test 4: Get User Profile"
echo "---------------------------"
PROFILE_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" $BASE_URL/me)
echo "Profile: $PROFILE_RESPONSE"

echo ""
echo "✅ Test 5: Search User by Member ID"
echo "-----------------------------------"
SEARCH_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/search/user?member_id=LBK001234")
echo "Search: $SEARCH_RESPONSE"

echo ""
echo "✅ Test 6: Transfer Points"
echo "-------------------------"
TRANSFER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"to_member_id":"LBK001234","amount":100}' \
  $BASE_URL/transfer)
echo "Transfer: $TRANSFER_RESPONSE"

echo ""
echo "✅ Test 7: Recent Transactions"
echo "------------------------------"
TRANSACTIONS_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" $BASE_URL/transactions/recent)
echo "Transactions: $TRANSACTIONS_RESPONSE"

echo ""
echo "❌ Test 8: Error Cases"
echo "----------------------"

echo "8.1 Invalid token:"
curl -s -H "Authorization: Bearer INVALID" $BASE_URL/me
echo ""

echo "8.2 Missing authorization:"
curl -s $BASE_URL/me
echo ""

echo "8.3 Transfer to non-existent user:"
curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"to_member_id":"INVALID123","amount":100}' \
  $BASE_URL/transfer
echo ""

echo "8.4 Transfer to self:"
curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"to_member_id\":\"$MEMBER_ID\",\"amount\":100}" \
  $BASE_URL/transfer
echo ""

echo "8.5 Transfer with insufficient balance:"
curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"to_member_id":"LBK001234","amount":999999}' \
  $BASE_URL/transfer
echo ""

echo ""
echo "✅ Test 9: Swagger Documentation"
echo "--------------------------------"
SWAGGER_RESPONSE=$(curl -s $BASE_URL/swagger/doc.json | head -c 200)
echo "Swagger API: $SWAGGER_RESPONSE..."

echo ""
echo "🎉 ALL TESTS COMPLETED!"
echo "======================="
