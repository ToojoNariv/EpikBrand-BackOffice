# Epik Brand Relook - Backend & Outil d'Administration Go

Ce projet est le backend d'administration pour **Epik Brand Relook**. Conçu en Go, il permet de gérer le portfolio dans une base de données PostgreSQL et de compiler l'application sous forme de fichier exécutable Windows (`.exe`).

## Fonctionnalités

1. **Auto-construction de la base de données** : Crée automatiquement la base PostgreSQL et la table `projects` si elles n'existent pas.
2. **Peuplement automatique (Seed)** : Insère automatiquement les 14 projets par défaut de la charte d'Epik Brand si la base est vide.
3. **Logiciel d'administration interactif (.exe)** : Permet d'ajouter des projets guidés pas à pas, de lister et de supprimer des projets.
4. **Serveur API REST** : Un serveur ultra-rapide servant les projets au format JSON attendu par Vue, incluant le support CORS.

---

## Configuration

Créez un fichier `.env` à la racine de ce dossier (ou copiez `.env.example`) :

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=votre_mot_de_passe
DB_NAME=epikbrand
DB_SSLMODE=disable

PORT=8080
```

---

## Utilisation en développement

1. Assurez-vous que PostgreSQL est démarré sur votre machine.
2. Pour lancer le logiciel interactif :
   ```bash
   go run main.go
   ```
3. Pour lancer uniquement le serveur API sans passer par le menu :
   ```bash
   go run main.go server
   ```

---

## Compilation en exécutable Windows (.exe)

Pour générer l'exécutable autonome `admin.exe` utilisable sous Windows :

```bash
go build -o admin.exe main.go
```

Une fois compilé, vous pouvez simplement **double-cliquer sur `admin.exe`** pour lancer la console interactive d'administration et gérer votre portfolio.

---

## Connexion avec le Front-End Vue.js

Pour connecter votre front-end Vue.js `Epik-Brand-Relook` à cette base de données dynamique, modifiez le fichier `src/components/Portfolio.vue` comme suit :

1. Remplacez le tableau statique `projetsParExpertise` par un `ref` vide ou réactif.
2. Récupérez les données via l'API lors du montage du composant.

### Exemple de modification dans `Portfolio.vue` :

```javascript
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue';

// 1. Déclarer une référence réactive pour les projets
const projetsParExpertise = ref({
  photo: [],
  video: [],
  graphique: [],
  web: []
});

// 2. Charger les projets depuis l'API Go
const chargerProjetsDepuisAPI = async () => {
  try {
    const reponse = await fetch('http://localhost:8080/api/projects');
    if (!reponse.ok) throw new Error("Erreur lors de la récupération des données");
    const projets = await reponse.json();

    // Réinitialiser les listes
    const structures = { photo: [], video: [], graphique: [], web: [] };
    
    // Répartir les projets par expertise
    projets.forEach(p => {
      if (structures[p.category]) {
        structures[p.category].push({
          id: p.id,
          titre: p.titre,
          mediaType: p.mediaType,
          src: p.src,
          bgColor: p.bgColor,
          description: p.description,
          galerie: p.galerie || []
        });
      }
    });

    projetsParExpertise.value = structures;
  } catch (erreur) {
    console.error("Impossible de joindre le backend Go, chargement des données locales de secours...", erreur);
  }
};

onMounted(() => {
  // Charger les projets au démarrage
  chargerProjetsDepuisAPI();
  
  // ... reste de votre code onMounted d'origine
});
```
