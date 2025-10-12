package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const videoDir = "./videos"

type URLRequest struct {
	URL string `json:"url"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

func main() {
	// Crée le dossier videos s'il n'existe pas
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Servir les vidéos en static
	fs := http.FileServer(http.Dir(videoDir))
	http.Handle("/videos/", http.StripPrefix("/videos/", fs))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/url", downloadURLHandler)
	http.HandleFunc("/urlyt", downloadYouTubeHandler)

	log.Println("Serveur démarré sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="fr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Serveur Vidéo</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 800px;
			margin: 50px auto;
			padding: 20px;
			background-color: #f5f5f5;
		}
		.container {
			background: white;
			padding: 30px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			margin-top: 0;
		}
		.form-group {
			margin-bottom: 20px;
		}
		label {
			display: block;
			margin-bottom: 5px;
			font-weight: bold;
			color: #555;
		}
		input[type="text"] {
			width: 100%;
			padding: 10px;
			border: 1px solid #ddd;
			border-radius: 4px;
			box-sizing: border-box;
			font-size: 14px;
		}
		button {
			padding: 12px 24px;
			margin-right: 10px;
			border: none;
			border-radius: 4px;
			cursor: pointer;
			font-size: 14px;
			font-weight: bold;
		}
		.btn-direct {
			background-color: #4CAF50;
			color: white;
		}
		.btn-youtube {
			background-color: #f44336;
			color: white;
		}
		button:hover {
			opacity: 0.9;
		}
		button:disabled {
			background-color: #ccc;
			cursor: not-allowed;
		}
		.message {
			margin-top: 20px;
			padding: 15px;
			border-radius: 4px;
			display: none;
		}
		.success {
			background-color: #d4edda;
			color: #155724;
			border: 1px solid #c3e6cb;
		}
		.error {
			background-color: #f8d7da;
			color: #721c24;
			border: 1px solid #f5c6cb;
		}
		.info {
			background-color: #d1ecf1;
			color: #0c5460;
			border: 1px solid #bee5eb;
		}
		.videos-link {
			margin-top: 20px;
			text-align: center;
		}
		.videos-link a {
			color: #007bff;
			text-decoration: none;
		}
		.videos-link a:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>📹 Téléchargeur de Vidéos</h1>
		
		<div class="form-group">
			<label for="url">URL de la vidéo:</label>
			<input type="text" id="url" placeholder="https://...">
		</div>
		
		<div>
			<button class="btn-direct" onclick="downloadDirect()">Télécharger (URL directe)</button>
			<button class="btn-youtube" onclick="downloadYouTube()">Télécharger (YouTube/yt-dlp)</button>
		</div>
		
		<div id="message" class="message"></div>
		
		<div class="videos-link">
			<a href="/videos/">📂 Voir toutes les vidéos</a>
		</div>
	</div>

	<script>
		const urlInput = document.getElementById('url');
		const messageDiv = document.getElementById('message');
		
		function showMessage(text, type) {
			messageDiv.textContent = text;
			messageDiv.className = 'message ' + type;
			messageDiv.style.display = 'block';
		}
		
		function hideMessage() {
			messageDiv.style.display = 'none';
		}
		
		async function download(endpoint, buttonText) {
			const url = urlInput.value.trim();
			
			if (!url) {
				showMessage('Veuillez entrer une URL', 'error');
				return;
			}
			
			hideMessage();
			const buttons = document.querySelectorAll('button');
			buttons.forEach(btn => btn.disabled = true);
			
			showMessage('Téléchargement en cours...', 'info');
			
			try {
				const response = await fetch(endpoint, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({ url: url })
				});
				
				const data = await response.json();
				
				if (data.success) {
					showMessage('✓ ' + data.message + (data.file ? ' - ' + data.file : ''), 'success');
					urlInput.value = '';
				} else {
					showMessage('✗ ' + data.message, 'error');
				}
			} catch (error) {
				showMessage('✗ Erreur: ' + error.message, 'error');
			} finally {
				buttons.forEach(btn => btn.disabled = false);
			}
		}
		
		function downloadDirect() {
			download('/url', 'Téléchargement direct');
		}
		
		function downloadYouTube() {
			download('/urlyt', 'Téléchargement YouTube');
		}
		
		// Permettre l'envoi avec Enter
		urlInput.addEventListener('keypress', function(e) {
			if (e.key === 'Enter') {
				downloadYouTube();
			}
		});
	</script>
</body>
</html>
	`))
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

	log.Printf("Téléchargement de: %s", req.URL)

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

	log.Printf("Vidéo téléchargée: %s", filename)

	sendSuccess(w, "Vidéo téléchargée avec succès", filename)
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

	log.Printf("Téléchargement YouTube de: %s", req.URL)

	// Nom de fichier pour yt-dlp
	outputTemplate := filepath.Join(videoDir, "%(title)s_%(id)s.%(ext)s")

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

	log.Printf("Vidéo YouTube téléchargée: %s", req.URL)
	log.Printf("Output: %s", output)

	sendSuccess(w, "Vidéo YouTube téléchargée avec succès", "")
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

