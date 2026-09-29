// Desktop: a interface é a da web (front/src), copiada sem alteração. Só este
// arquivo e src/desktop/ são próprios do Wails — a ponte troca o HTTP /api
// pelos bindings do PaintService e o download do navegador pelo diálogo nativo.
import './desktop/instalar';
import { mount } from 'svelte';
// D-001: tipografia e iconografia do protótipo Nocturne — Inter (única família)
// e Phosphor (classes `ph`/`ph-bold`), ambos self-hosted: o app desktop não
// depende de rede.
import '@fontsource-variable/inter';
import '@phosphor-icons/web/regular';
import '@phosphor-icons/web/bold';
import './app.css';
import App from './App.svelte';

const app = mount(App, { target: document.getElementById('app')! });

export default app;
