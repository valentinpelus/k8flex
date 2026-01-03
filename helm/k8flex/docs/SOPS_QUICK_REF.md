# SOPS Quick Reference Card

## 📋 Setup (One-time)

```bash
# Install tools
brew install sops age  # macOS
# OR
sudo apt install sops age  # Linux

# Run setup wizard
cd helm/k8flex
./setup-sops.sh

# Creates ~/.config/sops/age/keys.txt
# Updates .sops.yaml with your key
```

## 🔐 Working with Secrets

### Create & Encrypt
```bash
# 1. Copy example
cp secrets.example.yaml secrets.yaml

# 2. Fill with real values
vim secrets.yaml

# 3. Encrypt
sops --encrypt --in-place secrets.yaml
```

### Edit Encrypted Secrets
```bash
# Opens decrypted in editor
# Auto-encrypts on save
sops secrets.yaml
```

### View Decrypted
```bash
# View all
sops --decrypt secrets.yaml

# View specific value
sops --decrypt --extract '["slackSecrets"]["botToken"]' secrets.yaml
```

## 🚀 Deployment

### Simple (Recommended)
```bash
./deploy.sh
```

### helm-secrets Plugin
```bash
helm secrets upgrade k8flex . \
  --namespace k8flex \
  --values values.yaml \
  --values secrets.yaml
```

### Manual Decryption
```bash
sops --decrypt secrets.yaml | \
  helm upgrade k8flex . \
    --namespace k8flex \
    --values values.yaml \
    --values -
```

## 🔍 Verification

```bash
# Check if encrypted
grep -q "sops:" secrets.yaml && echo "✅ Encrypted" || echo "❌ NOT ENCRYPTED"

# Test decryption
sops --decrypt secrets.yaml > /dev/null && echo "✅ Can decrypt" || echo "❌ Cannot decrypt"

# Verify deployment
kubectl get pods -n k8flex
kubectl logs -n k8flex deployment/k8flex
```

## 👥 Team Sharing

### Add Team Member
```yaml
# .sops.yaml
creation_rules:
  - age: >-
      age1yourkey,
      age1teammatekey
```

```bash
# Update encrypted file with new recipient
sops updatekeys secrets.yaml
```

### Share Your Public Key
```bash
# Team member runs this
grep "public key:" ~/.config/sops/age/keys.txt

# Shares output: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7...
```

## 🔄 Rotation

```bash
# 1. Edit secrets
sops secrets.yaml

# 2. Rotate encryption
sops --rotate --in-place secrets.yaml

# 3. Deploy
./deploy.sh
```

## 🚨 Troubleshooting

### Cannot decrypt
```bash
# Check key file
echo $SOPS_AGE_KEY_FILE
cat $SOPS_AGE_KEY_FILE

# Set if missing
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt

# Add to ~/.zshrc or ~/.bashrc
echo 'export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt' >> ~/.zshrc
```

### File not encrypted
```bash
# Encrypt it
sops --encrypt --in-place secrets.yaml

# Verify
grep "sops:" secrets.yaml
```

### helm-secrets not found
```bash
helm plugin install https://github.com/jkroepke/helm-secrets
```

## 📁 File Structure

```
helm/k8flex/
├── .sops.yaml              # SOPS config (commit)
├── secrets.yaml            # Encrypted secrets (commit)
├── secrets.example.yaml    # Template (commit)
├── values.yaml            # Non-sensitive config (commit)
├── setup-sops.sh          # Setup wizard (commit)
└── deploy.sh              # Deployment script (commit)

~/.config/sops/age/
└── keys.txt               # Your private key (NEVER COMMIT)
```

## ✅ Checklist

Before first deployment:
- [ ] SOPS & age installed
- [ ] Ran `./setup-sops.sh`
- [ ] Created `secrets.yaml`
- [ ] Filled with real values
- [ ] Encrypted: `sops --encrypt --in-place secrets.yaml`
- [ ] Verified: `grep "sops:" secrets.yaml`
- [ ] Can decrypt: `sops --decrypt secrets.yaml`
- [ ] Removed secrets from `values.yaml`
- [ ] Tested: `./deploy.sh`

## 🔗 Links

- [Full Guide](SOPS_USAGE.md)
- [Migration Guide](../../SOPS_MIGRATION.md)
- [SOPS Docs](https://github.com/mozilla/sops)
- [age Docs](https://github.com/FiloSottile/age)

## 🎯 Quick Commands

```bash
# Setup
./setup-sops.sh

# Edit secrets
sops secrets.yaml

# Deploy
./deploy.sh

# View logs
kubectl logs -n k8flex deployment/k8flex -f

# That's it! 🎉
```
