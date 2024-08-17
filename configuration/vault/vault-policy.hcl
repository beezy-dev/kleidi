path "transit/encrypt/kleidi" {
   capabilities = [ "update" ]
}

path "transit/decrypt/kleidi" {
   capabilities = [ "update" ]
}

path "transit/keys/kleidi" {
   capabilities = [ "read" ]
}

path "auth/token/lookup-self" {
    capabilities = ["read"]
}

path "auth/token/renew-self" {
    capabilities = ["update"]
}