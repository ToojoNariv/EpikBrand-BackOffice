/* -----------------------------------------------------------------------------
 * COMPOSANTS LOGIQUES D'INTERFACE - ARCHITECTURE POUR ADMIN D'ENTITÉS
 * Ce code est conçu de façon modulaire et réutilisable pour d'autres projets.
 * -------------------------------------------------------------------------- */

// =============================================================================
// 1. SERVICES & API CLIENT
// =============================================================================
class ApiService {
  constructor(endpoint) {
    this.endpoint = endpoint;
  }

  getHeaders() {
    const token = localStorage.getItem('auth_token');
    const headers = {
      'Content-Type': 'application/json'
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  handleError(status, message) {
    if (status === 401) {
      console.warn("[API] Session expirée ou non autorisée. Déconnexion.");
      if (window.logoutUser) {
        window.logoutUser();
      } else {
        localStorage.removeItem('auth_token');
        location.reload();
      }
    }
    throw new Error(`${message}`);
  }

  async getAll() {
    const response = await fetch(this.endpoint, {
      headers: this.getHeaders()
    });
    if (!response.ok) {
      this.handleError(response.status, `Erreur de chargement`);
    }
    return response.json();
  }

  async save(payload) {
    const response = await fetch(this.endpoint, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      const msg = errorData.error || `Erreur lors de l'enregistrement`;
      this.handleError(response.status, msg);
    }
    return response.json();
  }

  async delete(id) {
    const response = await fetch(`${this.endpoint}?id=${id}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      const msg = errorData.error || `Erreur lors de la suppression`;
      this.handleError(response.status, msg);
    }
    return response.json();
  }
}

// =============================================================================
// 2. COMPOSANTS DE L'INTERFACE UTILISATEUR (UI COMPONENTS)
// =============================================================================

/**
 * Gère l'affichage, la fermeture et le cycle de vie des tiroirs (Drawers) / Modales.
 * Verrouille le défilement du corps principal à l'ouverture pour corriger l'ergonomie.
 */
class Drawer {
  constructor({ drawerId, overlayId, formId, titleId, defaultTitle }) {
    this.drawer = document.getElementById(drawerId);
    this.overlay = document.getElementById(overlayId);
    this.form = document.getElementById(formId);
    this.titleEl = document.getElementById(titleId);
    this.defaultTitle = defaultTitle;
    this.editingId = null;

    this.initCloseListeners();
  }

  initCloseListeners() {
    const btnClose = this.drawer.querySelector('.btn-close');
    const btnCancel = this.form.querySelector('.btn-light');

    if (btnClose) btnClose.addEventListener('click', () => this.close());
    if (btnCancel) btnCancel.addEventListener('click', () => this.close());
    if (this.overlay) this.overlay.addEventListener('click', () => this.close());
  }

  open(editingId = null, title = null) {
    this.editingId = editingId;
    this.titleEl.textContent = title || this.defaultTitle;
    
    this.drawer.classList.add('active');
    this.overlay.classList.add('active');
    document.body.style.overflow = 'hidden'; // Bloque le scroll sous-jacent
  }

  close() {
    this.editingId = null;
    this.drawer.classList.remove('active');
    this.overlay.classList.remove('active');
    document.body.style.overflow = ''; // Libère le scroll
  }

  reset() {
    if (this.form) this.form.reset();
    
    // Nettoyer les classes d'erreurs de validation
    this.form.querySelectorAll('.form-group').forEach(fg => {
      fg.classList.remove('has-error');
    });
  }
}

/**
 * Contrôle générique des onglets de traduction multilingue.
 */
class LangTabController {
  constructor(tabsContainerSelector, contentsPrefix) {
    this.container = document.querySelector(tabsContainerSelector);
    if (!this.container) return;

    this.tabs = this.container.querySelectorAll('.lang-tab');
    this.contentsPrefix = contentsPrefix;

    this.init();
  }

  init() {
    this.tabs.forEach(tab => {
      tab.addEventListener('click', () => {
        this.switchTab(tab.getAttribute('data-lang'));
      });
    });
  }

  switchTab(langCode) {
    // Activer l'onglet ciblé
    this.tabs.forEach(tab => {
      tab.classList.toggle('active', tab.getAttribute('data-lang') === langCode);
    });

    // Afficher le contenu de langue adéquat
    const allContents = document.querySelectorAll(`[id^="${this.contentsPrefix}"]`);
    allContents.forEach(content => {
      const suffix = content.id.endsWith(langCode);
      content.classList.toggle('active', suffix);
    });
  }
}

/**
 * Gère l'affichage en direct des aperçus d'images à partir d'un champ texte URL.
 */
class MediaPreviewController {
  constructor(inputId, previewBoxId, previewImgId) {
    this.input = document.getElementById(inputId);
    this.previewBox = document.getElementById(previewBoxId);
    this.previewImg = document.getElementById(previewImgId);

    if (this.input) {
      this.input.addEventListener('input', () => this.updatePreview());
    }
  }

  updatePreview() {
    const src = this.input.value.trim();
    if (src && (src.startsWith('http://') || src.startsWith('https://') || src.startsWith('/'))) {
      this.previewImg.src = src;
      this.previewBox.style.display = 'block';
    } else {
      this.hide();
    }
  }

  hide() {
    if (this.previewBox) {
      this.previewBox.style.display = 'none';
      this.previewImg.src = '';
    }
  }
}

/**
 * Gère la liste dynamique des médias de la galerie dans le formulaire.
 */
class GalleryController {
  constructor({ selectId, inputId, buttonId, listId }) {
    this.selectType = document.getElementById(selectId);
    this.inputSrc = document.getElementById(inputId);
    this.btnAdd = document.getElementById(buttonId);
    this.listContainer = document.getElementById(listId);
    this.items = [];

    if (this.btnAdd) {
      this.btnAdd.addEventListener('click', () => this.addItem());
    }
  }

  addItem() {
    const src = this.inputSrc.value.trim();
    const type = this.selectType.value;

    if (src === '') {
      this.inputSrc.focus();
      return;
    }

    this.items.push({ type, src });
    this.inputSrc.value = '';
    this.render();
  }

  setItems(items) {
    this.items = items ? [...items] : [];
    this.render();
  }

  getItems() {
    return this.items;
  }

  render() {
    if (this.items.length === 0) {
      this.listContainer.innerHTML = `<li class="gallery-empty">Aucun média secondaire dans ce projet.</li>`;
      return;
    }

    this.listContainer.innerHTML = '';
    this.items.forEach((item, index) => {
      const li = document.createElement('li');
      li.className = 'gallery-item';

      const mediaHTML = (item.type === 'video' && item.src)
        ? `<video src="${item.src}" autoplay loop muted playsinline class="gallery-item-media"></video>`
        : (item.src ? `<img src="${item.src}" class="gallery-item-media" alt="Aperçu" />` : `<div class="gallery-item-placeholder">Pas d'image</div>`);

      li.innerHTML = `
        ${mediaHTML}
        <button type="button" class="btn-remove-gallery" title="Enlever">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          </svg>
        </button>
      `;

      li.querySelector('.btn-remove-gallery').addEventListener('click', () => {
        this.items.splice(index, 1);
        this.render();
      });

      this.listContainer.appendChild(li);
    });
  }
}

// =============================================================================
// 3. APPLICATION PRINCIPALE
// =============================================================================
document.addEventListener('DOMContentLoaded', () => {
  
  // --- INSTANCIATIONS DE SERVICES ---
  const projectApi = new ApiService('/api/projects');
  const teamApi = new ApiService('/api/team');
  const userApi = new ApiService('/api/users');
  const settingsApi = new ApiService('/api/settings');

  // --- VARIABLES D'ÉTAT GLOBALES ---
  let projectsList = [];
  let teamList = [];
  let activeCategory = 'all';
  let currentActiveTab = 'portfolio';

  // --- CIBLES DE SUPPRESSION ---
  let deleteTargetId = null;
  let deleteTargetType = 'project';

  // --- ÉLÉMENTS DU DOM ---
  const projectsGrid = document.getElementById('projects-grid');
  const teamGrid = document.getElementById('team-grid');
  const navPortfolio = document.getElementById('nav-portfolio');
  const navEquipe = document.getElementById('nav-equipe');
  const navUsers = document.getElementById('nav-users');
  const sectionPortfolio = document.getElementById('section-portfolio');
  const sectionEquipe = document.getElementById('section-equipe');
  const sectionUsers = document.getElementById('section-users');
  const btnOpenForm = document.getElementById('btn-open-form');
  const btnOpenMemberForm = document.getElementById('btn-open-member-form');
  const btnOpenUserForm = document.getElementById('btn-open-user-form');
  const userTableBody = document.getElementById('users-table-body');
  const filterTabs = document.querySelectorAll('.filter-tab');

  // --- ÉLÉMENTS ANALYTICS ---
  const navAnalytics = document.getElementById('nav-analytics');
  const sectionAnalytics = document.getElementById('section-analytics');
  const analyticsEmptyState = document.getElementById('analytics-empty-state');
  const analyticsAdminHint = document.getElementById('analytics-admin-hint');
  const analyticsIframeWrapper = document.getElementById('analytics-iframe-wrapper');
  const analyticsIframe = document.getElementById('analytics-iframe');
  const analyticsWebviewHelper = document.getElementById('analytics-webview-helper');
  const btnAnalyticsGoogleLogin = document.getElementById('btn-analytics-google-login');
  const analyticsConfigCard = document.getElementById('analytics-config-card');
  const analyticsConfigForm = document.getElementById('analytics-config-form');
  const analyticsUrlInput = document.getElementById('analytics-url-input');

  // Dialog de confirmation
  const confirmModalOverlay = document.getElementById('confirm-modal-overlay');
  const confirmModal = document.getElementById('confirm-modal');
  const btnCancelDelete = document.getElementById('btn-cancel-delete');
  const btnConfirmDelete = document.getElementById('btn-confirm-delete');

  // Sélecteur de couleur lié
  const inputBgColor = document.getElementById('p-bg-color');
  const inputBgColorPicker = document.getElementById('p-bg-color-picker');
  
  if (inputBgColorPicker && inputBgColor) {
    inputBgColorPicker.addEventListener('input', (e) => {
      inputBgColor.value = e.target.value;
    });
    inputBgColor.addEventListener('input', (e) => {
      const val = e.target.value;
      if (val.startsWith('#') && val.length === 7) {
        inputBgColorPicker.value = val;
      }
    });
  }

  // --- INITIALISATIONS DES COMPOSANTS FRONT-END MODULAIRES ---
  const projectDrawer = new Drawer({
    drawerId: 'project-drawer',
    overlayId: 'drawer-overlay',
    formId: 'project-form',
    titleId: 'form-title',
    defaultTitle: 'Ajouter un Projet'
  });

  const memberDrawer = new Drawer({
    drawerId: 'member-drawer',
    overlayId: 'member-drawer-overlay',
    formId: 'member-form',
    titleId: 'member-form-title',
    defaultTitle: 'Ajouter un Membre'
  });

  const projectLangTabs = new LangTabController('#project-drawer .lang-tabs', 'lang-content-');
  const memberLangTabs = new LangTabController('#member-lang-tabs', 'member-lang-content-');
  
  const coverPreview = new MediaPreviewController('p-src', 'cover-preview-box', 'cover-preview-img');
  
  const galleryController = new GalleryController({
    selectId: 'g-type',
    inputId: 'g-src',
    buttonId: 'btn-add-gallery',
    listId: 'gallery-list'
  });

  // --- VALIDATEUR DE FORMULAIRE ---
  const validateFormGroup = (inputEl) => {
    const formGroup = inputEl.closest('.form-group');
    if (!formGroup) return true;
    
    if (inputEl.value.trim() === '') {
      formGroup.classList.add('has-error');
      return false;
    } else {
      formGroup.classList.remove('has-error');
      return true;
    }
  };

  // --- RENDU ET CHARGEMENT DU PORTFOLIO (PROJETS) ---
  const renderProjects = () => {
    const filtered = activeCategory === 'all' 
      ? projectsList 
      : projectsList.filter(p => p.category === activeCategory);
      
    if (!filtered || filtered.length === 0) {
      projectsGrid.innerHTML = `
        <div class="no-projects">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
            <line x1="8" y1="21" x2="16" y2="21"></line>
            <line x1="12" y1="17" x2="12" y2="21"></line>
          </svg>
          <p>Aucun projet dans cette catégorie.</p>
        </div>
      `;
      return;
    }

    projectsGrid.innerHTML = '';
    
    filtered.forEach(p => {
      let fullTitle = p.title_fr_part1;
      if (p.title_fr_part2) {
        fullTitle += ' ' + p.title_fr_part2;
      }
      
      const card = document.createElement('article');
      card.className = 'project-card';
      
      const mediaHTML = p.mediaType === 'video'
        ? `<video src="${p.src}" autoplay loop muted playsinline class="card-media"></video>`
        : `<img src="${p.src}" alt="${fullTitle}" class="card-media" loading="lazy">`;

      card.innerHTML = `
        <div class="card-media-wrapper">
          ${mediaHTML}
          
          <div class="project-tag-bleu">
            ${fullTitle}
          </div>
          
          <div class="card-actions">
            <button type="button" class="btn-action btn-action-edit" title="Modifier ce projet" data-id="${p.id}">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                <path d="M18.5 2.5a2.121 2.121 0 1 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
              </svg>
            </button>
            <button type="button" class="btn-action btn-action-delete" title="Supprimer ce projet" data-id="${p.id}" data-title="${fullTitle}">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                <line x1="10" y1="11" x2="10" y2="17"></line>
                <line x1="14" y1="11" x2="14" y2="17"></line>
              </svg>
            </button>
          </div>
        </div>
        
        <div class="card-footer">
          <span class="card-category">${p.category}</span>
          <span class="card-id">${p.id}</span>
        </div>
      `;

      // Clic actions
      card.querySelector('.btn-action-edit').addEventListener('click', (e) => {
        e.stopPropagation();
        openEditProject(p);
      });

      card.querySelector('.btn-action-delete').addEventListener('click', (e) => {
        e.stopPropagation();
        openDeleteModal(p.id, 'project');
      });

      projectsGrid.appendChild(card);
    });
  };

  const loadProjects = async () => {
    projectsGrid.innerHTML = `
      <div class="loader-container">
        <div class="loader"></div>
        <p>Chargement des projets depuis PostgreSQL...</p>
      </div>
    `;
    try {
      projectsList = await projectApi.getAll();
      renderProjects();
    } catch (error) {
      console.error(error);
      projectsGrid.innerHTML = `
        <div class="no-projects">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"></polygon>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          <p>Impossible de charger le portfolio. La base de données PostgreSQL est-elle connectée ?</p>
          <p style="font-size: 0.8rem; margin-top: 0.5rem; color: var(--color-danger);">${error.message}</p>
        </div>
      `;
    }
  };

  // --- RENDU ET CHARGEMENT DE L'ÉQUIPE ---
  const renderTeam = () => {
    if (!teamList || teamList.length === 0) {
      teamGrid.innerHTML = `
        <div class="no-projects">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
            <line x1="8" y1="21" x2="16" y2="21"></line>
            <line x1="12" y1="17" x2="12" y2="21"></line>
          </svg>
          <p>Aucun membre dans l'équipe pour le moment.</p>
        </div>
      `;
      return;
    }

    teamGrid.innerHTML = '';
    
    teamList.forEach(m => {
      const fullName = `${m.prenom} ${m.nom}`;
      const displayRole = m.role_fr || m.role || "Membre";

      const card = document.createElement('article');
      card.className = 'project-card';

      card.innerHTML = `
        <div class="card-media-wrapper">
          <img src="${m.photo}" alt="${fullName}" class="card-media" loading="lazy">
          
          <div class="project-tag-bleu">
            ${fullName}
          </div>
          
          <div class="card-actions">
            <button type="button" class="btn-action btn-action-edit" title="Modifier ce membre" data-id="${m.id}">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                <path d="M18.5 2.5a2.121 2.121 0 1 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
              </svg>
            </button>
            <button type="button" class="btn-action btn-action-delete" title="Supprimer ce membre" data-id="${m.id}" data-title="${fullName}">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                <line x1="10" y1="11" x2="10" y2="17"></line>
                <line x1="14" y1="11" x2="14" y2="17"></line>
              </svg>
            </button>
          </div>
        </div>
        
        <div class="card-footer">
          <span class="card-category">${displayRole}</span>
          <span class="card-id" style="font-size: 0.7rem;">${m.email}</span>
        </div>
      `;

      // Clic actions
      card.querySelector('.btn-action-edit').addEventListener('click', (e) => {
        e.stopPropagation();
        openEditMember(m);
      });

      card.querySelector('.btn-action-delete').addEventListener('click', (e) => {
        e.stopPropagation();
        openDeleteModal(m.id, 'member');
      });

      teamGrid.appendChild(card);
    });
  };

  const loadTeam = async () => {
    teamGrid.innerHTML = `
      <div class="loader-container">
        <div class="loader"></div>
        <p>Chargement des membres de l'équipe...</p>
      </div>
    `;
    try {
      teamList = await teamApi.getAll();
      renderTeam();
    } catch (error) {
      console.error(error);
      teamGrid.innerHTML = `
        <div class="no-projects">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"></polygon>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          <p>Impossible de charger l'équipe. La base de données PostgreSQL est-elle connectée ?</p>
          <p style="font-size: 0.8rem; margin-top: 0.5rem; color: var(--color-danger);">${error.message}</p>
        </div>
      `;
    }
  };

  // --- ACTIONS OUVERTURE DRAWER MODIFIER ---
  const openEditProject = (p) => {
    projectDrawer.reset();
    
    // Remplir les valeurs dans le drawer
    document.getElementById('p-category').value = p.category || 'photo';
    document.getElementById('p-media-type').value = p.mediaType || 'image';
    
    const inputSrc = document.getElementById('p-src');
    inputSrc.value = p.src || '';
    
    const pBgColor = p.bgColor || '#0096E7';
    document.getElementById('p-bg-color').value = pBgColor;
    document.getElementById('p-bg-color-picker').value = pBgColor;

    // Remplir les traductions
    document.getElementById('p-title-fr-1').value = p.title_fr_part1 || '';
    document.getElementById('p-title-fr-2').value = p.title_fr_part2 || '';
    document.getElementById('p-desc-fr').value = p.description_fr || '';

    document.getElementById('p-title-en-1').value = p.title_en_part1 || '';
    document.getElementById('p-title-en-2').value = p.title_en_part2 || '';
    document.getElementById('p-desc-en').value = p.description_en || '';

    document.getElementById('p-title-mg-1').value = p.title_mg_part1 || '';
    document.getElementById('p-title-mg-2').value = p.title_mg_part2 || '';
    document.getElementById('p-desc-mg').value = p.description_mg || '';

    // Déclencher l'aperçu image
    coverPreview.updatePreview();

    // Remplir la galerie
    galleryController.setItems(p.galerie);

    // Initialiser les onglets de langue
    projectLangTabs.switchTab('fr');

    // Ouvrir le Drawer
    projectDrawer.open(p.id, "Modifier le Projet");
  };

  const openEditMember = (m) => {
    memberDrawer.reset();

    // Remplir les valeurs
    document.getElementById('m-prenom').value = m.prenom || '';
    document.getElementById('m-nom').value = m.nom || '';
    document.getElementById('m-email').value = m.email || '';
    document.getElementById('m-photo').value = m.photo || '';
    document.getElementById('m-role-fr').value = m.role_fr || '';
    document.getElementById('m-role-en').value = m.role_en || '';
    document.getElementById('m-role-mg').value = m.role_mg || '';

    // Initialiser les onglets de langue
    memberLangTabs.switchTab('fr');

    // Ouvrir le Drawer
    memberDrawer.open(m.id, "Modifier le Membre");
  };

  // --- ACTIONS OUVERTURE DRAWER AJOUTER ---
  btnOpenForm.addEventListener('click', () => {
    projectDrawer.reset();
    coverPreview.hide();
    galleryController.setItems([]);
    projectLangTabs.switchTab('fr');
    projectDrawer.open(null, "Ajouter un Projet");
  });

  btnOpenMemberForm.addEventListener('click', () => {
    memberDrawer.reset();
    memberLangTabs.switchTab('fr');
    memberDrawer.open(null, "Ajouter un Membre");
  });

  // --- REQUÊTE ET SOUMISSION DU FORMULAIRE PROJET ---
  const projectForm = document.getElementById('project-form');
  projectForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    let isValid = true;
    const inputSrc = document.getElementById('p-src');
    const pBgColor = document.getElementById('p-bg-color');
    const titleFr1 = document.getElementById('p-title-fr-1');
    const descFr = document.getElementById('p-desc-fr');

    // Valider les champs requis
    if (!validateFormGroup(inputSrc)) isValid = false;
    if (!validateFormGroup(pBgColor)) isValid = false;
    
    const isFrTitleOk = validateFormGroup(titleFr1);
    const isFrDescOk = validateFormGroup(descFr);
    if (!isFrTitleOk || !isFrDescOk) isValid = false;

    if (!isValid) {
      if (!isFrTitleOk || !isFrDescOk) {
        projectLangTabs.switchTab('fr');
      }
      return;
    }

    const titleFr2 = document.getElementById('p-title-fr-2').value.trim();
    
    // Anglais (repli vers Français si vide)
    let titleEn1 = document.getElementById('p-title-en-1').value.trim();
    let titleEn2 = document.getElementById('p-title-en-2').value.trim();
    let descEn = document.getElementById('p-desc-en').value.trim();
    if (titleEn1 === '') {
      titleEn1 = titleFr1.value.trim();
      titleEn2 = titleFr2;
    }
    if (descEn === '') {
      descEn = descFr.value.trim();
    }

    // Malgache (repli vers Français si vide)
    let titleMg1 = document.getElementById('p-title-mg-1').value.trim();
    let titleMg2 = document.getElementById('p-title-mg-2').value.trim();
    let descMg = document.getElementById('p-desc-mg').value.trim();
    if (titleMg1 === '') {
      titleMg1 = titleFr1.value.trim();
      titleMg2 = titleFr2;
    }
    if (descMg === '') {
      descMg = descFr.value.trim();
    }

    const payload = {
      id: projectDrawer.editingId || "",
      category: document.getElementById('p-category').value,
      mediaType: document.getElementById('p-media-type').value,
      src: inputSrc.value.trim(),
      bgColor: pBgColor.value.trim(),
      galerie: galleryController.getItems(),
      
      // FR
      title_fr_part1: titleFr1.value.trim(),
      title_fr_part2: titleFr2,
      description_fr: descFr.value.trim(),
      
      // EN
      title_en_part1: titleEn1,
      title_en_part2: titleEn2,
      description_en: descEn,
      
      // MG
      title_mg_part1: titleMg1,
      title_mg_part2: titleMg2,
      description_mg: descMg
    };

    try {
      await projectApi.save(payload);
      projectDrawer.close();
      loadProjects();
    } catch (error) {
      alert(`Erreur lors de l'enregistrement : ${error.message}`);
    }
  });

  // --- REQUÊTE ET SOUMISSION DU FORMULAIRE MEMBRE ---
  const memberForm = document.getElementById('member-form');
  memberForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    let isValid = true;
    const prenom = document.getElementById('m-prenom');
    const nom = document.getElementById('m-nom');
    const email = document.getElementById('m-email');
    const photo = document.getElementById('m-photo');
    const roleFr = document.getElementById('m-role-fr');

    if (!validateFormGroup(prenom)) isValid = false;
    if (!validateFormGroup(nom)) isValid = false;
    if (!validateFormGroup(email)) isValid = false;
    if (!validateFormGroup(photo)) isValid = false;
    
    const isRoleFrOk = validateFormGroup(roleFr);
    if (!isRoleFrOk) isValid = false;

    if (!isValid) {
      if (!isRoleFrOk) {
        memberLangTabs.switchTab('fr');
      }
      return;
    }

    let roleEn = document.getElementById('m-role-en').value.trim();
    if (roleEn === '') roleEn = roleFr.value.trim();

    let roleMg = document.getElementById('m-role-mg').value.trim();
    if (roleMg === '') roleMg = roleFr.value.trim();

    const payload = {
      id: memberDrawer.editingId || "",
      prenom: prenom.value.trim(),
      nom: nom.value.trim(),
      email: email.value.trim(),
      photo: photo.value.trim(),
      role_fr: roleFr.value.trim(),
      role_en: roleEn,
      role_mg: roleMg
    };

    try {
      await teamApi.save(payload);
      memberDrawer.close();
      loadTeam();
    } catch (error) {
      alert(`Erreur lors de l'enregistrement : ${error.message}`);
    }
  });

  // --- MODAL DE CONFIRMATION DE SUPPRESSION ---
  const openDeleteModal = (id, type) => {
    deleteTargetId = id;
    deleteTargetType = type;
    confirmModalOverlay.classList.add('active');
    confirmModal.classList.add('active');
  };

  const closeDeleteModal = () => {
    deleteTargetId = null;
    confirmModalOverlay.classList.remove('active');
    confirmModal.classList.remove('active');
    
    // Rétablir les textes d'origine
    confirmModal.querySelector('.confirm-header h3').textContent = "Confirmer la suppression";
    confirmModal.querySelector('.confirm-body p').textContent = "Êtes-vous sûr de vouloir supprimer définitivement cet élément ? Cette action est irréversible.";
    btnConfirmDelete.textContent = "Supprimer";
  };

  btnCancelDelete.addEventListener('click', closeDeleteModal);
  confirmModalOverlay.addEventListener('click', closeDeleteModal);

  btnConfirmDelete.addEventListener('click', async () => {
    if (!deleteTargetId) return;

    try {
      if (deleteTargetType === 'project') {
        await projectApi.delete(deleteTargetId);
        loadProjects();
        closeDeleteModal();
      } else if (deleteTargetType === 'member') {
        await teamApi.delete(deleteTargetId);
        loadTeam();
        closeDeleteModal();
      } else if (deleteTargetType === 'user') {
        await userApi.delete(deleteTargetId);
        loadUsers();
        closeDeleteModal();
      } else if (deleteTargetType === 'transfer-admin') {
        const token = localStorage.getItem('auth_token');
        const res = await fetch('/api/users/transfer-admin', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          },
          body: JSON.stringify({ user_id: deleteTargetId })
        });
        const data = await res.json();
        if (!res.ok) {
          throw new Error(data.error || "Erreur de transfert");
        }
        alert("Transfert du rôle d'administrateur effectué avec succès. Vous allez être déconnecté.");
        window.logoutUser();
        closeDeleteModal();
      }
    } catch (error) {
      alert(`Erreur : ${error.message}`);
      closeDeleteModal();
    }
  });

  // --- NAVIGATION PAR ONGLETS (PORTFOLIO / ÉQUIPE / MODÉRATEURS) ---
  navPortfolio.addEventListener('click', () => {
    navPortfolio.classList.add('active');
    navEquipe.classList.remove('active');
    navUsers.classList.remove('active');
    navAnalytics.classList.remove('active');
    
    sectionPortfolio.style.display = 'block';
    sectionEquipe.style.display = 'none';
    sectionUsers.style.display = 'none';
    sectionAnalytics.style.display = 'none';
    
    btnOpenForm.style.display = 'inline-flex';
    btnOpenMemberForm.style.display = 'none';
    btnOpenUserForm.style.display = 'none';
    
    currentActiveTab = 'portfolio';
    loadProjects();
  });

  navEquipe.addEventListener('click', () => {
    navPortfolio.classList.remove('active');
    navEquipe.classList.add('active');
    navUsers.classList.remove('active');
    navAnalytics.classList.remove('active');
    
    sectionPortfolio.style.display = 'none';
    sectionEquipe.style.display = 'block';
    sectionUsers.style.display = 'none';
    sectionAnalytics.style.display = 'none';
    
    btnOpenForm.style.display = 'none';
    btnOpenMemberForm.style.display = 'inline-flex';
    btnOpenUserForm.style.display = 'none';
    
    currentActiveTab = 'equipe';
    loadTeam();
  });

  // --- FILTRES DE CATÉGORIE ---
  filterTabs.forEach(tab => {
    tab.addEventListener('click', () => {
      filterTabs.forEach(t => t.classList.remove('active'));
      tab.classList.add('active');
      activeCategory = tab.getAttribute('data-category');
      renderProjects();
    });
  });

  const fetchMe = async () => {
    const token = localStorage.getItem('auth_token');
    if (!token) return;
    try {
      const res = await fetch('/api/auth/me', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        localStorage.setItem('user_email', data.email);
        localStorage.setItem('user_name', data.name);
        localStorage.setItem('user_role', data.role);
        localStorage.setItem('user_picture_url', data.picture_url || '');
        
        // Mettre à jour l'affichage en direct
        document.getElementById('user-display-name').textContent = data.name;
        document.getElementById('user-display-role').textContent = data.role === 'admin' ? 'Administrateur' : 'Modérateur';
        const avatarImg = document.getElementById('user-display-avatar');
        if (avatarImg) {
          if (data.picture_url) {
            avatarImg.src = data.picture_url;
            avatarImg.style.display = 'block';
          } else {
            avatarImg.style.display = 'none';
          }
        }
      } else if (res.status === 401) {
        window.logoutUser();
      }
    } catch (e) {
      console.warn("Impossible de rafraîchir le profil utilisateur :", e);
    }
  };

  // --- GESTION DES SESSIONS ---
  const checkSession = () => {
    const token = localStorage.getItem('auth_token');
    const email = localStorage.getItem('user_email');
    const name = localStorage.getItem('user_name');
    const role = localStorage.getItem('user_role');
    const pictureUrl = localStorage.getItem('user_picture_url');

    if (token && role) {
      document.getElementById('login-overlay').classList.remove('active');
      document.getElementById('header-profile').style.display = 'flex';
      document.getElementById('user-display-name').textContent = name;
      document.getElementById('user-display-role').textContent = role === 'admin' ? 'Administrateur' : 'Modérateur';
      
      const avatarImg = document.getElementById('user-display-avatar');
      if (avatarImg) {
        if (pictureUrl) {
          avatarImg.src = pictureUrl;
          avatarImg.style.display = 'block';
        } else {
          avatarImg.style.display = 'none';
        }
      }
      
      if (role === 'admin') {
        navUsers.style.display = 'inline-block';
      } else {
        navUsers.style.display = 'none';
      }

      navAnalytics.style.display = 'inline-block';

      loadProjects();
      fetchMe();
    } else {
      document.getElementById('login-overlay').classList.add('active');
      document.getElementById('header-profile').style.display = 'none';
      navUsers.style.display = 'none';
      navAnalytics.style.display = 'none';
      
      // Réinitialisation de la navigation
      sectionPortfolio.style.display = 'block';
      sectionEquipe.style.display = 'none';
      sectionUsers.style.display = 'none';
      sectionAnalytics.style.display = 'none';
      
      navPortfolio.classList.add('active');
      navEquipe.classList.remove('active');
      navUsers.classList.remove('active');
      navAnalytics.classList.remove('active');
      
      btnOpenForm.style.display = 'inline-flex';
      btnOpenMemberForm.style.display = 'none';
      btnOpenUserForm.style.display = 'none';
      currentActiveTab = 'portfolio';
    }
  };

  window.logoutUser = async () => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${token}` }
        });
      } catch (e) {
        console.warn("Échec de la déconnexion serveur :", e);
      }
    }
    localStorage.removeItem('auth_token');
    localStorage.removeItem('user_email');
    localStorage.removeItem('user_name');
    localStorage.removeItem('user_role');
    localStorage.removeItem('user_picture_url');
    checkSession();
  };

  document.getElementById('btn-logout').addEventListener('click', window.logoutUser);

  // --- CONNEXION GOOGLE SIGN-IN ---
  const initGoogleSignIn = async () => {
    const waitForGoogle = () => {
      return new Promise((resolve) => {
        if (window.google && window.google.accounts) return resolve();
        const interval = setInterval(() => {
          if (window.google && window.google.accounts) {
            clearInterval(interval);
            resolve();
          }
        }, 50);
      });
    };

    try {
      await waitForGoogle();
      const response = await fetch('/api/config');
      if (!response.ok) throw new Error("Impossible de charger la configuration Google");
      const config = await response.json();
      
      if (!config.google_client_id) {
        console.error("GOOGLE_CLIENT_ID manquant sur le serveur.");
        document.getElementById('login-error').style.display = 'block';
        document.getElementById('login-error').textContent = "Configuration serveur incomplète : GOOGLE_CLIENT_ID manquant.";
        return;
      }

      google.accounts.id.initialize({
        client_id: config.google_client_id,
        callback: handleCredentialResponse
      });
      google.accounts.id.renderButton(
        document.getElementById("google-signin-btn"),
        { theme: "outline", size: "large", text: "signin_with" }
      );
    } catch (err) {
      console.error("Erreur initialisation Google Sign-in:", err);
      document.getElementById('login-error').style.display = 'block';
      document.getElementById('login-error').textContent = "Erreur lors de la connexion au serveur d'authentification.";
    }
  };

  const handleCredentialResponse = async (response) => {
    document.getElementById('login-error').style.display = 'none';
    try {
      const res = await fetch('/api/auth/google', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: response.credential })
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Erreur d'authentification");
      }
      
      localStorage.setItem('auth_token', data.token);
      localStorage.setItem('user_email', data.email);
      localStorage.setItem('user_name', data.name);
      localStorage.setItem('user_role', data.role);
      localStorage.setItem('user_picture_url', data.picture_url || '');
      
      checkSession();
    } catch (err) {
      console.error(err);
      document.getElementById('login-error').style.display = 'block';
      document.getElementById('login-error').textContent = err.message;
    }
  };

  // --- GESTION DES MODÉRATEURS (ADMIN SEULEMENT) ---

  const loadUsers = async () => {
    userTableBody.innerHTML = `
      <tr>
        <td colspan="6" style="text-align: center; color: var(--text-muted); padding: 2rem;">
          Chargement des modérateurs...
        </td>
      </tr>
    `;
    try {
      const usersList = await userApi.getAll();
      renderUsers(usersList);
    } catch (err) {
      console.error(err);
      userTableBody.innerHTML = `
        <tr>
          <td colspan="6" style="text-align: center; color: var(--color-danger); padding: 2rem;">
            Impossible de charger les modérateurs : ${err.message}
          </td>
        </tr>
      `;
    }
  };

  const renderUsers = (users) => {
    if (!users || users.length === 0) {
      userTableBody.innerHTML = `
        <tr>
          <td colspan="6" style="text-align: center; color: var(--text-muted); padding: 2rem;">
            Aucun modérateur enregistré.
          </td>
        </tr>
      `;
      return;
    }

    userTableBody.innerHTML = '';
    const currentEmail = localStorage.getItem('user_email');
    
    users.forEach(u => {
      const tr = document.createElement('tr');
      const isSelf = u.email === currentEmail;
      const roleText = u.role === 'admin' ? '<strong>Administrateur</strong>' : 'Modérateur';
      
      let actionsHTML = '';
      if (u.role === 'moderator') {
        actionsHTML = `
          <div class="user-actions-group">
            <button class="btn-action-solid btn-transfer-admin" data-id="${u.id}" data-name="${u.name}">
              Nommer admin
            </button>
            <button class="btn-action-solid btn-delete-user" data-id="${u.id}" data-name="${u.name}">
              Supprimer
            </button>
          </div>
        `;
      } else {
        actionsHTML = `<span style="font-size: 0.85rem; color: var(--text-muted); font-style: italic;">Propriétaire</span>`;
      }

      let avatarHTML = '';
      if (u.picture_url) {
        avatarHTML = `<img class="user-table-avatar" src="${u.picture_url}" alt="${u.name}" referrerpolicy="no-referrer" />`;
      } else {
        const initials = u.name ? u.name.split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase() : '?';
        avatarHTML = `<div class="user-table-avatar-placeholder">${initials}</div>`;
      }

      tr.innerHTML = `
        <td>${avatarHTML}</td>
        <td>${u.name} ${isSelf ? ' <span class="user-role-badge">Vous</span>' : ''}</td>
        <td style="font-family: monospace;">${u.email}</td>
        <td>${roleText}</td>
        <td style="font-size: 0.8rem; color: var(--text-muted);">${new Date(u.created_at).toLocaleDateString('fr-FR')}</td>
        <td style="text-align: right;">${actionsHTML}</td>
      `;

      const btnDelete = tr.querySelector('.btn-delete-user');
      if (btnDelete) {
        btnDelete.addEventListener('click', () => openDeleteUserModal(u.id, `${u.name} (${u.email})`));
      }

      const btnTransfer = tr.querySelector('.btn-transfer-admin');
      if (btnTransfer) {
        btnTransfer.addEventListener('click', () => openTransferAdminModal(u.id, `${u.name} (${u.email})`));
      }

      userTableBody.appendChild(tr);
    });
  };

  // Navigation onglet modérateurs
  navUsers.addEventListener('click', () => {
    navPortfolio.classList.remove('active');
    navEquipe.classList.remove('active');
    navUsers.classList.add('active');
    navAnalytics.classList.remove('active');
    
    sectionPortfolio.style.display = 'none';
    sectionEquipe.style.display = 'none';
    sectionUsers.style.display = 'block';
    sectionAnalytics.style.display = 'none';
    
    btnOpenForm.style.display = 'none';
    btnOpenMemberForm.style.display = 'none';
    btnOpenUserForm.style.display = 'inline-flex';
    
    currentActiveTab = 'users';
    loadUsers();
  });

  // Navigation onglet Statistiques (Looker Studio)
  navAnalytics.addEventListener('click', () => {
    navPortfolio.classList.remove('active');
    navEquipe.classList.remove('active');
    navUsers.classList.remove('active');
    navAnalytics.classList.add('active');
    
    sectionPortfolio.style.display = 'none';
    sectionEquipe.style.display = 'none';
    sectionUsers.style.display = 'none';
    sectionAnalytics.style.display = 'block';
    
    btnOpenForm.style.display = 'none';
    btnOpenMemberForm.style.display = 'none';
    btnOpenUserForm.style.display = 'none';
    
    currentActiveTab = 'analytics';
    loadAnalytics();
  });

  const loadAnalytics = async () => {
    analyticsEmptyState.style.display = 'none';
    analyticsIframeWrapper.style.display = 'none';
    analyticsIframe.src = '';
    
    const isWebView2 = window.chrome && window.chrome.webview;
    if (isWebView2) {
      analyticsWebviewHelper.style.display = 'block';
    } else {
      analyticsWebviewHelper.style.display = 'none';
    }
    
    const role = localStorage.getItem('user_role');
    if (role === 'admin') {
      analyticsConfigCard.style.display = 'block';
      analyticsAdminHint.style.display = 'block';
    } else {
      analyticsConfigCard.style.display = 'none';
      analyticsAdminHint.style.display = 'none';
    }

    try {
      const res = await fetch('/api/settings?key=looker_studio_url');
      if (!res.ok) throw new Error("Erreur de récupération");
      const data = await res.json();
      const url = data.value || '';

      if (url.trim() === '') {
        analyticsEmptyState.style.display = 'flex';
      } else {
        analyticsIframe.src = url;
        analyticsIframeWrapper.style.display = 'block';
      }

      if (role === 'admin') {
        analyticsUrlInput.value = url;
      }
    } catch (err) {
      console.error(err);
      analyticsEmptyState.style.display = 'flex';
    }
  };

  if (btnAnalyticsGoogleLogin) {
    btnAnalyticsGoogleLogin.addEventListener('click', () => {
      window.location.href = "https://accounts.google.com/ServiceLogin?continue=https%3A%2F%2Flookerstudio.google.com%2F";
    });
  }

  // Enregistrement de la configuration Looker Studio
  analyticsConfigForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const urlVal = analyticsUrlInput.value.trim();
    const token = localStorage.getItem('auth_token');

    try {
      const res = await fetch('/api/settings', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          key: 'looker_studio_url',
          value: urlVal
        })
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Erreur de configuration");
      }
      alert("Configuration de Looker Studio enregistrée avec succès.");
      loadAnalytics();
    } catch (err) {
      alert(`Erreur : ${err.message}`);
    }
  });

  // Drawer Modérateur
  const userDrawer = new Drawer({
    drawerId: 'user-drawer',
    overlayId: 'user-drawer-overlay',
    formId: 'user-form',
    titleId: 'user-form-title',
    defaultTitle: 'Ajouter un Modérateur'
  });

  btnOpenUserForm.addEventListener('click', () => {
    userDrawer.reset();
    userDrawer.open(null, "Ajouter un Modérateur");
  });

  const btnCloseUserForm = document.getElementById('btn-close-user-form');
  const btnCancelUserForm = document.getElementById('btn-cancel-user-form');
  if (btnCloseUserForm) btnCloseUserForm.addEventListener('click', () => userDrawer.close());
  if (btnCancelUserForm) btnCancelUserForm.addEventListener('click', () => userDrawer.close());

  const userForm = document.getElementById('user-form');
  userForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const uName = document.getElementById('u-name');
    const uEmail = document.getElementById('u-email');
    
    let isValid = true;
    if (!validateFormGroup(uName)) isValid = false;
    if (!validateFormGroup(uEmail)) isValid = false;
    
    if (!isValid) return;

    try {
      await userApi.save({
        name: uName.value.trim(),
        email: uEmail.value.trim()
      });
      userDrawer.close();
      loadUsers();
    } catch (err) {
      alert(`Erreur : ${err.message}`);
    }
  });

  const openDeleteUserModal = (id, name) => {
    deleteTargetId = id;
    deleteTargetType = 'user';
    
    const confirmModalTitle = confirmModal.querySelector('.confirm-header h3');
    const confirmModalBody = confirmModal.querySelector('.confirm-body p');
    
    confirmModalTitle.textContent = "Supprimer le modérateur";
    confirmModalBody.textContent = `Êtes-vous sûr de vouloir supprimer définitivement le modérateur "${name}" ? Cet utilisateur ne pourra plus se connecter.`;
    
    confirmModalOverlay.classList.add('active');
    confirmModal.classList.add('active');
  };

  const openTransferAdminModal = (id, name) => {
    deleteTargetId = id;
    deleteTargetType = 'transfer-admin';
    
    const confirmModalTitle = confirmModal.querySelector('.confirm-header h3');
    const confirmModalBody = confirmModal.querySelector('.confirm-body p');
    
    confirmModalTitle.textContent = "Transférer le rôle Administrateur";
    confirmModalBody.innerHTML = `
      <strong>ATTENTION :</strong> Vous allez transférer le rôle d'administrateur principal à <strong>${name}</strong>.
      <br><br>
      Une fois validé, <strong>vous deviendrez modérateur</strong> et perdrez les droits de gestion des utilisateurs.
      Vous serez automatiquement déconnecté pour que le nouveau rôle s'applique.
    `;
    
    btnConfirmDelete.textContent = "Transférer";
    
    confirmModalOverlay.classList.add('active');
    confirmModal.classList.add('active');
  };

  // --- CHARGEMENT INITIAL ---
  checkSession();
  initGoogleSignIn();
});
