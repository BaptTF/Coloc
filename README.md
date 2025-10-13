# 📹 Video Server - Serveur de Téléchargement de Vidéos

Un serveur web en Go pour télécharger des vidéos avec support VLC et auto-play.

## 🚀 Fonctionnalités

- **Téléchargement direct** depuis des URLs
- **Téléchargement YouTube** avec yt-dlp
- **Authentification VLC** sécurisée côté serveur
- **Auto-play** intelligent après téléchargement
- **Sessions persistantes** VLC
- **Interface web** moderne et responsive

## 🐳 Déploiement Docker

### Prérequis
- Docker & Docker Compose installés
- Serveur VLC accessible avec interface web activée

### Installation rapide

```bash
# Cloner le projet
git clone <repository>
cd video-server

# Construire et lancer
docker-compose up -d

# Vérifier le statut
docker-compose ps
```

### Configuration VLC

1. **Activer l'interface web VLC** :
   ```bash
   vlc --intf http --http-password your-password
   ```

2. **Ou via interface** : `Outils` → `Préférences` → `Interface` → `Interface web`

### Utilisation

1. **Accéder à l'interface** : http://localhost:8080
2. **Configurer VLC** : Entrer l'URL de votre serveur VLC
3. **S'authentifier** : Cliquer "Se connecter" et entrer le code à 4 chiffres
4. **Télécharger** : Coller une URL et choisir le mode de téléchargement

## 📋 Commandes Docker

### Gestion des services

```bash
# Démarrer
docker-compose up -d

# Arrêter
docker-compose down

# Redémarrer
docker-compose restart

# Voir les logs
docker-compose logs -f video-server

# Rebuild après modifications
docker-compose build --no-cache
docker-compose up -d
```

### Gestion des volumes

```bash
# Nettoyer les vidéos
rm -rf ./videos/*

# Backup des vidéos
tar -czf videos-backup.tar.gz videos/

# Restore des vidéos
tar -xzf videos-backup.tar.gz
```

## 🔧 Configuration

### Variables d'environnement

Modifiez `docker-compose.yml` :

```yaml
environment:
  - GIN_MODE=release      # Mode de production
  - LOG_LEVEL=info        # Niveau de log
  - PORT=8080            # Port d'écoute
```

### Volumes persistants

- `./videos:/app/videos` - Stockage des vidéos téléchargées
- `./logs:/app/logs` - Logs de l'application (optionnel)

### Limites de ressources

```yaml
deploy:
  resources:
    limits:
      memory: 512M
      cpus: '0.5'
```

## 🛠️ Développement local

### Sans Docker

```bash
# Installer yt-dlp
python3 -m pip install -U --pre "yt-dlp[default]"

# Compiler et lancer
go build -o video-server main.go
./video-server
```

### Avec Docker (dev)

```bash
# Build de développement
docker build -t video-server:dev .

# Lancer en mode développement
docker run -p 8080:8080 \
  -v $(pwd)/videos:/app/videos \
  video-server:dev
```

## 📊 Monitoring

### Health check

Le conteneur inclut un health check automatique :

```bash
# Vérifier la santé
docker-compose ps
```

### Logs structurés

Les logs sont en format JSON pour faciliter le monitoring :

```bash
docker-compose logs video-server | jq '.'
```

## 🔒 Sécurité

### Bonnes pratiques implémentées

- ✅ Image FROM scratch ultra-minimale (~15MB)
- ✅ Binaire Go statique sans dépendances
- ✅ yt-dlp binaire statique (pas d'interpréteur Python)
- ✅ Multi-stage build pour image optimale
- ✅ Authentification VLC sécurisée côté serveur
- ✅ Pas d'exposition des cookies VLC au client
- ✅ Validation des entrées utilisateur

### Recommandations

1. **Proxy inverse** : Utiliser nginx ou Traefik
2. **HTTPS** : Activer TLS en production
3. **Réseau** : Isoler dans un réseau Docker dédié
4. **Backup** : Sauvegarder régulièrement les vidéos

## 🐛 Dépannage

### Problèmes courants

**VLC inaccessible** :
```bash
# Vérifier la connectivité
curl http://your-vlc-server:8080/

# Vérifier les logs VLC
docker-compose logs video-server | grep VLC
```

**Téléchargement échoue** :
```bash
# Tester yt-dlp manuellement
docker-compose exec video-server yt-dlp --version

# Vérifier les permissions
docker-compose exec video-server ls -la /app/videos/
```

**Problème de mémoire** :
```bash
# Augmenter les limites dans docker-compose.yml
memory: 1G
```

## 🔄 Mise à jour

```bash
# Arrêter le service
docker-compose down

# Mettre à jour le code
git pull

# Rebuild et relancer
docker-compose build --no-cache
docker-compose up -d
```

## 📄 Licence

Ce projet est sous licence MIT.
