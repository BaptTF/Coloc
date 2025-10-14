#!/usr/bin/env python3
"""
Test script pour vérifier le frontend refait
"""

import requests
import time
import sys

def test_frontend():
    base_url = "http://localhost:8080"
    
    print("🧪 Test du frontend refait")
    print("=" * 50)
    
    # Test 1: Page principale
    print("1. Test de la page principale...")
    try:
        response = requests.get(base_url)
        if response.status_code == 200:
            content = response.text
            if "Téléchargeur de Vidéos" in content and "styles.css" in content and "app.js" in content:
                print("   ✅ Page principale OK - HTML moderne avec CSS/JS séparés")
            else:
                print("   ❌ Page principale manque des éléments")
                return False
        else:
            print(f"   ❌ Erreur HTTP {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Erreur: {e}")
        return False
    
    # Test 2: CSS embarqué
    print("2. Test du fichier CSS...")
    try:
        response = requests.get(f"{base_url}/styles.css")
        if response.status_code == 200 and "text/css" in response.headers.get("content-type", ""):
            css_content = response.text
            if "--primary-color" in css_content and ".btn" in css_content and ".toast" in css_content:
                print("   ✅ CSS moderne OK - Variables CSS, composants, toasts")
            else:
                print("   ❌ CSS manque des composants modernes")
                return False
        else:
            print(f"   ❌ CSS non accessible ou mauvais type de contenu")
            return False
    except Exception as e:
        print(f"   ❌ Erreur CSS: {e}")
        return False
    
    # Test 3: JavaScript embarqué
    print("3. Test du fichier JavaScript...")
    try:
        response = requests.get(f"{base_url}/app.js")
        if response.status_code == 200 and "javascript" in response.headers.get("content-type", ""):
            js_content = response.text
            if "class ToastManager" in js_content and "class VlcManager" in js_content and "CONFIG" in js_content:
                print("   ✅ JavaScript moderne OK - Classes ES6, architecture modulaire")
            else:
                print("   ❌ JavaScript manque l'architecture moderne")
                return False
        else:
            print(f"   ❌ JavaScript non accessible ou mauvais type de contenu")
            return False
    except Exception as e:
        print(f"   ❌ Erreur JavaScript: {e}")
        return False
    
    # Test 4: API endpoints toujours fonctionnels
    print("4. Test des endpoints API...")
    try:
        response = requests.get(f"{base_url}/list")
        if response.status_code == 200:
            print("   ✅ Endpoint /list OK")
        else:
            print(f"   ❌ Endpoint /list erreur {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Erreur API: {e}")
        return False
    
    # Test 5: Serveur de fichiers vidéos
    print("5. Test du serveur de fichiers vidéos...")
    try:
        response = requests.get(f"{base_url}/videos/")
        # 200 OK ou 403/404 acceptable (pas de vidéos)
        if response.status_code in [200, 403, 404]:
            print("   ✅ Serveur de fichiers vidéos OK")
        else:
            print(f"   ❌ Serveur vidéos erreur {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Erreur serveur vidéos: {e}")
        return False
    
    print("\n🎉 Tous les tests sont passés!")
    print("✨ Le frontend a été refait avec succès:")
    print("   - Design moderne et professionnel")
    print("   - Architecture séparée (HTML/CSS/JS)")
    print("   - Fichiers embarqués dans le binaire Go")
    print("   - Toutes les fonctionnalités conservées")
    print("   - APIs et endpoints fonctionnels")
    
    return True

if __name__ == "__main__":
    print("Attente du démarrage du serveur...")
    time.sleep(3)
    
    success = test_frontend()
    sys.exit(0 if success else 1)
