# Using SOPS with k8flex Helm Chart

This guide explains how to use Mozilla SOPS to encrypt sensitive values in the k8flex Helm chart.

## What is SOPS?

[SOPS](https://github.com/mozilla/sops) (Secrets OPerationS) is a tool for managing encrypted secrets in version control. It supports multiple encryption backends:
- **age** - Simple, modern encryption (recommended for local/simple setups)
- **AWS KMS** - AWS Key Management Service (recommended for EKS)
- **GCP KMS** - Google Cloud KMS (recommended for GKE)
- **Azure Key Vault** - Azure Key Vault (recommended for AKS)
- **PGP** - Traditional GPG encryption

## Installation

### macOS
```bash
brew install sops age
```

### Linux
```bash
# SOPS
wget https://github.com/mozilla/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64
sudo mv sops-v3.8.1.linux.amd64 /usr/local/bin/sops
sudo chmod +x /usr/local/bin/sops

# age
sudo apt install age
```

### Windows
```powershell
# Using Chocolatey
choco install sops age
```

## Setup

### Option 1: Age Encryption (Recommended for Getting Started)

1. **Generate age key**:
```bash
mkdir -p ~/.config/sops/age
age-keygen -o ~/.config/sops/age/keys.txt
```

2. **Note your public key** (shown in the output):
```
Public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

3. **Update `.sops.yaml`**:
```yaml
creation_rules:
  - path_regex: secrets\.yaml$
    encrypted_regex: ^(llmSecrets|slackSecrets|webhookSecrets|knowledgeBaseSecrets)$
    age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

4. **Set environment variable** (add to ~/.bashrc or ~/.zshrc):
```bash
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
```

### Option 2: AWS KMS (Recommended for EKS)

1. **Create KMS key**:
```bash
aws kms create-key --description "k8flex SOPS encryption key"
```

2. **Note the Key ID** from the output

3. **Update `.sops.yaml`**:
```yaml
creation_rules:
  - path_regex: secrets\.yaml$
    encrypted_regex: ^(llmSecrets|slackSecrets|webhookSecrets|knowledgeBaseSecrets)$
    kms: arn:aws:kms:us-east-1:123456789012:key/YOUR-KEY-ID
```

4. **Ensure AWS credentials are configured**:
```bash
aws configure
```

### Option 3: GCP KMS (Recommended for GKE)

1. **Create KMS keyring and key**:
```bash
gcloud kms keyrings create sops --location global
gcloud kms keys create sops-key --location global --keyring sops --purpose encryption
```

2. **Update `.sops.yaml`**:
```yaml
creation_rules:
  - path_regex: secrets\.yaml$
    encrypted_regex: ^(llmSecrets|slackSecrets|webhookSecrets|knowledgeBaseSecrets)$
    gcp_kms: projects/YOUR-PROJECT/locations/global/keyRings/sops/cryptoKeys/sops-key
```

## Usage

### 1. Fill in Secrets

Edit `secrets.yaml` with your actual secrets:
```yaml
llmSecrets:
  openaiApiKey: "sk-proj-your-actual-key"
  anthropicApiKey: "sk-ant-your-actual-key"
  geminiApiKey: "AIza-your-actual-key"

slackSecrets:
  botToken: "xoxb-your-actual-token"
  webhookUrl: "https://hooks.slack.com/services/..."

webhookSecrets:
  authToken: "your-actual-auth-token"

knowledgeBaseSecrets:
  databaseUrl: "postgresql://user:pass@host:5432/db?sslmode=require"
  embeddingApiKey: ""
```

### 2. Encrypt Secrets

```bash
cd helm/k8flex
sops --encrypt --in-place secrets.yaml
```

The file will now be encrypted (safe to commit):
```yaml
llmSecrets:
    openaiApiKey: ENC[AES256_GCM,data:encrypted_data_here,iv:...,tag:...,type:str]
    # ... encrypted values
sops:
    kms: []
    gcp_kms: []
    azure_kv: []
    age:
        - recipient: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
          enc: |
            -----BEGIN AGE ENCRYPTED FILE-----
            ...
```

### 3. Deploy with Helm

Using helm-secrets plugin (recommended):

```bash
# Install helm-secrets plugin
helm plugin install https://github.com/jkroepke/helm-secrets

# Deploy with secrets
helm secrets upgrade k8flex . \
  --namespace k8flex \
  --create-namespace \
  --values values.yaml \
  --values secrets.yaml
```

**OR** using manual decryption:

```bash
# Decrypt and deploy
sops --decrypt secrets.yaml | helm upgrade k8flex . \
  --namespace k8flex \
  --create-namespace \
  --values values.yaml \
  --values -
```

**OR** pre-decrypt (less secure):

```bash
# Decrypt to temporary file
sops --decrypt secrets.yaml > secrets.dec.yaml

# Deploy
helm upgrade k8flex . \
  --namespace k8flex \
  --values values.yaml \
  --values secrets.dec.yaml

# Clean up immediately
rm secrets.dec.yaml
```

### 4. Edit Encrypted Secrets

SOPS makes editing encrypted files easy:

```bash
sops secrets.yaml
```

This opens your editor with decrypted content. When you save and exit, SOPS automatically re-encrypts.

### 5. View Encrypted Secrets

```bash
# View decrypted content
sops --decrypt secrets.yaml

# View specific value
sops --decrypt --extract '["slackSecrets"]["botToken"]' secrets.yaml
```

## GitOps with ArgoCD/Flux

### ArgoCD with SOPS

1. **Install SOPS plugin**:
```yaml
# argocd-cm ConfigMap
data:
  configManagementPlugins: |
    - name: helm-sops
      generate:
        command: [sh, -c]
        args:
          - |
            helm secrets template . \
              --name-template $ARGOCD_APP_NAME \
              --namespace $ARGOCD_APP_NAMESPACE \
              --values values.yaml \
              --values secrets.yaml
```

2. **Create age key secret**:
```bash
kubectl create secret generic sops-age \
  --namespace argocd \
  --from-file=keys.txt=$HOME/.config/sops/age/keys.txt
```

3. **Mount in ArgoCD repo-server**:
```yaml
# argocd-repo-server deployment
env:
  - name: SOPS_AGE_KEY_FILE
    value: /sops/keys.txt
volumeMounts:
  - name: sops-age
    mountPath: /sops
volumes:
  - name: sops-age
    secret:
      secretName: sops-age
```

### Flux CD with SOPS

1. **Create age key secret**:
```bash
cat ~/.config/sops/age/keys.txt | kubectl create secret generic sops-age \
  --namespace flux-system \
  --from-file=age.agekey=/dev/stdin
```

2. **Configure kustomization**:
```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: k8flex
  namespace: flux-system
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-age
  sourceRef:
    kind: GitRepository
    name: k8flex
  path: ./helm/k8flex
```

## Team Sharing

### Multiple Recipients (Age)

To allow multiple team members to decrypt:

1. **Collect public keys** from team members
2. **Update `.sops.yaml`**:
```yaml
creation_rules:
  - path_regex: secrets\.yaml$
    encrypted_regex: ^(llmSecrets|slackSecrets|webhookSecrets|knowledgeBaseSecrets)$
    age: >-
      age1teammate1pubkey,
      age1teammate2pubkey,
      age1teammate3pubkey
```

3. **Re-encrypt with new recipients**:
```bash
sops updatekeys secrets.yaml
```

### Cloud KMS (Automatic Team Access)

When using AWS KMS, GCP KMS, or Azure Key Vault, access is controlled by IAM:
- Grant team members access to the KMS key
- They can decrypt using their cloud credentials
- No need to share keys directly

## Best Practices

1. **Never commit unencrypted secrets**:
   ```bash
   # Add to .gitignore
   secrets.dec.yaml
   secrets.yaml.dec
   ```

2. **Always verify encryption before commit**:
   ```bash
   # Check if file is encrypted
   grep -q "sops:" secrets.yaml && echo "✅ Encrypted" || echo "❌ NOT ENCRYPTED"
   ```

3. **Use `.sops.yaml` in repository**:
   - Ensures consistent encryption
   - Team members don't need to remember encryption commands

4. **Rotate secrets regularly**:
   ```bash
   # 1. Edit secrets
   sops secrets.yaml
   
   # 2. Re-encrypt with rotation
   sops --rotate --in-place secrets.yaml
   
   # 3. Deploy new secrets
   helm secrets upgrade k8flex . --values secrets.yaml
   ```

5. **Backup encryption keys**:
   - Store age keys securely (password manager, secure backup)
   - Document KMS key ARNs
   - Keep recovery plan for key loss

6. **Use different keys per environment**:
   ```
   helm/k8flex/
     .sops.yaml
     values.yaml
     secrets-dev.yaml      # Encrypted with dev key
     secrets-staging.yaml  # Encrypted with staging key
     secrets-prod.yaml     # Encrypted with prod key
   ```

## Troubleshooting

### "Failed to get the data key"
- Ensure `SOPS_AGE_KEY_FILE` is set and points to valid key file
- For KMS: Check AWS/GCP/Azure credentials and permissions

### "MAC mismatch"
- File was manually edited after encryption
- Re-edit with `sops secrets.yaml` to fix

### "No master key was able to decrypt"
- Your key is not in the recipients list
- Ask admin to add your public key: `sops updatekeys secrets.yaml`

### helm-secrets not found
```bash
helm plugin list
helm plugin install https://github.com/jkroepke/helm-secrets
```

## Migration from values.yaml

If you have secrets in `values.yaml`:

1. **Copy sensitive values to `secrets.yaml`**
2. **Encrypt**: `sops --encrypt --in-place secrets.yaml`
3. **Remove from `values.yaml`** (replace with empty strings)
4. **Test deployment** with both files
5. **Commit encrypted `secrets.yaml`**

## Security Notes

- ✅ Encrypted secrets are safe to commit to git
- ✅ SOPS only encrypts values, not keys (readable structure)
- ✅ Encrypted with AES256-GCM (industry standard)
- ⚠️ Protect your encryption keys (age keys, KMS access)
- ⚠️ Don't share decrypted files via insecure channels
- ⚠️ Regularly rotate both secrets and encryption keys

## Learn More

- [SOPS Documentation](https://github.com/mozilla/sops)
- [helm-secrets Plugin](https://github.com/jkroepke/helm-secrets)
- [age Documentation](https://github.com/FiloSottile/age)
- [ArgoCD with SOPS](https://argo-cd.readthedocs.io/en/stable/operator-manual/secret-management/)
- [Flux with SOPS](https://fluxcd.io/flux/guides/mozilla-sops/)
