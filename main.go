package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

//go:embed index.html
var indexHTML []byte

//go:embed styles.css
var stylesCSS []byte

//go:embed app.js
var appJS []byte

const videoDir = "./videos"

// Structure pour maintenir les sessions VLC
type VLCSession struct {
	Challenge     string
	Client        *http.Client
	URL           string
	Authenticated bool
	LastActivity  time.Time
}

// Map pour stocker les sessions VLC par URL
var vlcSessions = make(map[string]*VLCSession)
var vlcSessionsMutex sync.RWMutex

// Configuration VLC persistante
type VLCConfig struct {
	URL           string `json:"url"`
	Authenticated bool   `json:"authenticated"`
	LastActivity  string `json:"last_activity"`
}

type URLRequest struct {
	URL        string `json:"url"`
	AutoPlay   bool   `json:"autoPlay,omitempty"`
	VLCUrl     string `json:"vlcUrl,omitempty"`
	BackendUrl string `json:"backendUrl,omitempty"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

func main() {
	// Configure logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// Crée le dossier videos s'il n'existe pas
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		logrus.Fatal(err)
	}

	// Servir les vidéos en static
	fs := http.FileServer(http.Dir(videoDir))
	http.Handle("/videos/", http.StripPrefix("/videos/", fs))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/styles.css", stylesHandler)
	http.HandleFunc("/app.js", appHandler)
	http.HandleFunc("/url", downloadURLHandler)
	http.HandleFunc("/urlyt", downloadYouTubeHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/vlc/code", vlcCodeHandler)
	http.HandleFunc("/vlc/verify-code", vlcVerifyHandler)
	http.HandleFunc("/vlc/play", vlcPlayHandler)
	http.HandleFunc("/vlc/status", vlcStatusHandler)
	http.HandleFunc("/vlc/config", vlcConfigHandler)

	logrus.Info("Serveur démarré sur http://localhost:8080")
	logrus.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func stylesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(stylesCSS)
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(appJS)
}

// downloadURLHandler télécharge une vidéo depuis une URL directe
func downloadURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req URLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		sendError(w, "URL manquante", http.StatusBadRequest)
		return
	}

	logrus.WithField("url", req.URL).Info("Début de téléchargement direct")

	// Télécharger le fichier
	resp, err := http.Get(req.URL)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur de téléchargement: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sendError(w, fmt.Sprintf("Erreur HTTP: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	// Générer un nom de fichier unique
	filename := fmt.Sprintf("video_%d.mp4", time.Now().Unix())
	filePath := filepath.Join(videoDir, filename)

	// Créer le fichier
	out, err := os.Create(filePath)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur de création du fichier: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Copier le contenu
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur d'écriture: %v", err), http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"url":      req.URL,
		"autoplay": req.AutoPlay,
	}).Info("Vidéo téléchargée avec succès")

	// Auto-play si demandé
	if req.AutoPlay && req.VLCUrl != "" && req.BackendUrl != "" {
		go autoPlayVideo(filename, req.VLCUrl, req.BackendUrl)
	}

	sendSuccess(w, "Vidéo téléchargée avec succès", filename)
	pruneVideos()
}

// downloadYouTubeHandler télécharge une vidéo avec yt-dlp
func downloadYouTubeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req URLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		sendError(w, "URL manquante", http.StatusBadRequest)
		return
	}

	logrus.WithField("url", req.URL).Info("Début de téléchargement YouTube")

	// Nom de fichier pour yt-dlp
	outputTemplate := filepath.Join(videoDir, "%(title)s.%(ext)s")

	// Check if yt-dlp is updated
	cmd := exec.Command("./yt-dlp", "-U")
	output, err := cmd.CombinedOutput()
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur yt-dlp -U: %v\n%s", err, output), http.StatusInternalServerError)
		return
	}

	// Appeler yt-dlp
	cmd = exec.Command("./yt-dlp",
		"-f", "best[ext=mp4]",
		"-o", outputTemplate,
		"--no-playlist",
		req.URL,
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur yt-dlp: %v\n%s", err, output), http.StatusInternalServerError)
		return
	}

	// Méthode plus fiable: trouver le fichier le plus récemment modifié
	var newFileName string
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		sendError(w, "Erreur lecture dossier videos après téléchargement", http.StatusInternalServerError)
		return
	}

	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newFileName = entry.Name()
		}
	}

	logrus.WithFields(logrus.Fields{
		"url":      req.URL,
		"output":   string(output),
		"new_file": newFileName,
		"autoplay": req.AutoPlay,
	}).Info("Vidéo YouTube téléchargée avec succès")

	// Auto-play si demandé
	if req.AutoPlay && req.VLCUrl != "" && req.BackendUrl != "" && newFileName != "" {
		go autoPlayVideo(newFileName, req.VLCUrl, req.BackendUrl)
	}

	sendSuccess(w, "Vidéo YouTube téléchargée avec succès", newFileName)
	pruneVideos()
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: message,
	})
}

func sendSuccess(w http.ResponseWriter, message string, filename string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		File:    filename,
	})
}

// verifyVideoAccessible vérifie qu'une vidéo est accessible via HTTP avant de lancer VLC
func verifyVideoAccessible(videoPath string, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		logrus.WithFields(logrus.Fields{
			"video_path":  videoPath,
			"attempt":     i + 1,
			"max_retries": maxRetries,
		}).Info("AUTO-PLAY - Vérification accessibilité vidéo")

		resp, err := http.Head(videoPath)
		if err == nil && resp.StatusCode == http.StatusOK {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
			}).Info("AUTO-PLAY - Vidéo accessible via HTTP")
			return true
		}

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
				"error":      err.Error(),
			}).Warn("AUTO-PLAY - Erreur vérification HTTP")
		} else {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
				"status":     resp.StatusCode,
			}).Warn("AUTO-PLAY - Vidéo pas encore accessible")
		}

		// Attendre avant le prochain essai (backoff exponentiel)
		waitTime := time.Duration(500*(i+1)) * time.Millisecond
		time.Sleep(waitTime)
	}

	logrus.WithFields(logrus.Fields{
		"video_path":  videoPath,
		"max_retries": maxRetries,
	}).Error("AUTO-PLAY - Vidéo toujours pas accessible après tous les essais")
	return false
}

// autoPlayVideo lance automatiquement une vidéo sur VLC si une session authentifiée existe
func autoPlayVideo(filename string, vlcUrl string, backendUrl string) {
	if filename == "" || vlcUrl == "" {
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
	}).Info("AUTO-PLAY - Tentative de lecture automatique")

	// Vérifier si une session VLC authentifiée existe
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcUrl).Warn("AUTO-PLAY - Pas de session VLC authentifiée")
		return
	}

	// Construire l'URL de la vidéo avec proper path encoding
	// Utiliser PathEscape pour les chemins URL (pas QueryEscape)
	encodedFilename := url.PathEscape(filename)
	videoPath := backendUrl + "/videos/" + encodedFilename

	// Vérifier que la vidéo est accessible via HTTP avant de contacter VLC
	// Extended wait time for debugging timing issues
	if !verifyVideoAccessible(videoPath, 60) {
		logrus.WithFields(logrus.Fields{
			"filename":   filename,
			"video_path": videoPath,
		}).Error("AUTO-PLAY - Vidéo non accessible, annulation auto-play")
		return
	}

	baseUrl, _ := url.Parse(vlcUrl + "/play")
	queryParams := baseUrl.Query()
	queryParams.Set("id", "-1")
	queryParams.Set("path", videoPath)
	queryParams.Set("type", "stream")
	baseUrl.RawQuery = queryParams.Encode()
	playUrl := baseUrl.String()

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("AUTO-PLAY - Envoi commande lecture à VLC")

	// Créer la requête
	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		logrus.WithError(err).Error("AUTO-PLAY - Erreur création requête")
		return
	}

	// Utiliser le client de la session pour maintenir l'authentification
	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("AUTO-PLAY - Erreur connexion VLC")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"vlc_url":  vlcUrl,
		}).Info("AUTO-PLAY - Lecture automatique réussie")
	} else {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"vlc_url":  vlcUrl,
			"status":   resp.StatusCode,
		}).Warn("AUTO-PLAY - Échec lecture automatique")
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		sendError(w, "Impossible de lister les vidéos", http.StatusInternalServerError)
		return
	}

	// Structure pour trier par date de modification
	type fileWithTime struct {
		name    string
		modTime time.Time
	}

	var filesWithTime []fileWithTime
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			filesWithTime = append(filesWithTime, fileWithTime{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// Trier par date de modification (plus récent en premier)
	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.After(filesWithTime[j].modTime)
	})

	// Extraire juste les noms de fichiers
	var files []string
	for _, f := range filesWithTime {
		files = append(files, f.name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func pruneVideos() {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		logrus.WithError(err).Error("Erreur pruneVideos")
		return
	}
	type fileInfo struct {
		name    string
		modTime time.Time
	}
	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
	}
	if len(files) <= 10 {
		return
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})
	for _, fi := range files[:len(files)-10] {
		os.Remove(filepath.Join(videoDir, fi.name))
	}
}

func vlcCodeHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC CODE - Début demande challenge")

	// Créer un client avec jar de cookies pour maintenir la session
	jar, err := cookiejar.New(nil)
	if err != nil {
		sendError(w, "Erreur création cookie jar", http.StatusInternalServerError)
		return
	}
	client := &http.Client{Jar: jar}

	// Selon test.py, il faut faire un POST avec form data: challenge=""
	formData := url.Values{}
	formData.Set("challenge", "")
	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"data":    formData.Encode(),
	}).Info("VLC CODE - Envoi vers VLC")

	resp, err := client.Post(vlcUrl+"/code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC CODE - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC CODE - Status reçu de VLC")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	challenge := string(body)
	logrus.WithFields(logrus.Fields{
		"vlc_url":          vlcUrl,
		"challenge":        challenge,
		"challenge_length": len(challenge),
	}).Info("VLC CODE - Challenge reçu de VLC")

	// Stocker la session pour cette URL VLC
	vlcSessionsMutex.Lock()
	vlcSessions[vlcUrl] = &VLCSession{
		Challenge: challenge,
		Client:    client,
		URL:       vlcUrl,
	}
	vlcSessionsMutex.Unlock()

	logrus.WithField("vlc_url", vlcUrl).Info("VLC CODE - Session stockée")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func vlcVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC VERIFY - Début vérification code")

	// Récupérer la session VLC stockée
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists {
		logrus.WithField("vlc_url", vlcUrl).Error("VLC VERIFY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez redemander un code", http.StatusBadRequest)
		return
	}

	// Parser le JSON du client pour extraire le code brut (4 chiffres)
	var clientData map[string]string
	if err := json.NewDecoder(r.Body).Decode(&clientData); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC VERIFY - Erreur parsing JSON")
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":     vlcUrl,
		"client_data": clientData,
	}).Info("VLC VERIFY - Données reçues du client")

	rawCode, exists := clientData["code"]
	if !exists {
		logrus.WithFields(logrus.Fields{
			"vlc_url":     vlcUrl,
			"client_data": clientData,
		}).Error("VLC VERIFY - Code manquant dans les données")
		sendError(w, "Code manquant", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":   vlcUrl,
		"raw_code":  rawCode,
		"challenge": session.Challenge,
	}).Info("VLC VERIFY - Code brut reçu du client")

	// Calculer le hash côté serveur comme dans test.py: sha256(code + challenge)
	hasher := sha256.New()
	hasher.Write([]byte(rawCode + session.Challenge))
	hashBytes := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)

	logrus.WithFields(logrus.Fields{
		"vlc_url":       vlcUrl,
		"raw_code":      rawCode,
		"challenge":     session.Challenge,
		"concatenation": rawCode + session.Challenge,
		"hash":          hashHex,
		"hash_length":   len(hashHex),
	}).Info("VLC VERIFY - Hash calculé côté serveur")

	// Selon test.py, VLC attend form data: code=<hash>
	formData := url.Values{}
	formData.Set("code", hashHex)

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"data":    formData.Encode(),
	}).Info("VLC VERIFY - Envoi vers VLC")

	resp, err := session.Client.Post(vlcUrl+"/verify-code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC VERIFY -  coErreurnnexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC VERIFY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"response": string(respBody),
	}).Info("VLC VERIFY - Réponse VLC")

	// Si l'authentification réussit, mettre à jour la session
	if resp.StatusCode == http.StatusOK {
		vlcSessionsMutex.Lock()
		session.Authenticated = true
		session.LastActivity = time.Now()
		vlcSessionsMutex.Unlock()
		logrus.WithField("vlc_url", vlcUrl).Info("VLC VERIFY - Authentification réussie, session maintenue")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// vlcStatusHandler retourne l'état d'authentification pour une URL VLC
func vlcStatusHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	config := VLCConfig{
		URL:           vlcUrl,
		Authenticated: exists && session.Authenticated,
	}

	if exists {
		config.LastActivity = session.LastActivity.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// vlcConfigHandler gère la configuration VLC (GET pour récupérer, POST pour définir)
func vlcConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Récupérer toutes les sessions VLC actives
		vlcSessionsMutex.RLock()
		configs := make([]VLCConfig, 0, len(vlcSessions))
		for url, session := range vlcSessions {
			configs = append(configs, VLCConfig{
				URL:           url,
				Authenticated: session.Authenticated,
				LastActivity:  session.LastActivity.Format(time.RFC3339),
			})
		}
		vlcSessionsMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs)

	case http.MethodPost:
		// Définir une nouvelle URL VLC (sans authentification)
		var config VLCConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			sendError(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if config.URL == "" {
			sendError(w, "URL VLC manquante", http.StatusBadRequest)
			return
		}

		// Créer une nouvelle session non authentifiée
		vlcSessionsMutex.Lock()
		vlcSessions[config.URL] = &VLCSession{
			URL:           config.URL,
			Authenticated: false,
			LastActivity:  time.Now(),
		}
		vlcSessionsMutex.Unlock()

		logrus.WithField("vlc_url", config.URL).Info("VLC CONFIG - Nouvelle URL VLC définie")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Message: "URL VLC définie",
		})

	default:
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

func vlcPlayHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC PLAY - Début lecture vidéo")

	// Récupérer la session VLC stockée
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists {
		logrus.WithField("vlc_url", vlcUrl).Error("VLC PLAY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez vous authentifier", http.StatusUnauthorized)
		return
	}

	// Construire l'URL avec tous les paramètres (sans le paramètre vlc)
	playUrl := vlcUrl + "/play?" + r.URL.RawQuery
	// Retirer le paramètre vlc de l'URL
	if queryParams := r.URL.Query(); len(queryParams) > 0 {
		queryParams.Del("vlc")
		playUrl = vlcUrl + "/play?" + queryParams.Encode()
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("VLC PLAY - URL construite")

	// Créer une nouvelle requête en utilisant le client de la session
	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		sendError(w, "Erreur création requête", http.StatusInternalServerError)
		return
	}

	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC PLAY - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC PLAY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"response": string(respBody),
	}).Info("VLC PLAY - Réponse VLC")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}
