import { mount } from 'svelte'
import App from './App.svelte'
import 'svelte-material-ui/bare.css'
import '@fontsource-variable/archivo/wdth.css'
import '@fontsource-variable/archivo/wdth-italic.css'
import '@fontsource-variable/bricolage-grotesque/standard.css'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/500.css'
import '@fontsource/ibm-plex-mono/600.css'
import './app.css'

mount(App, { target: document.getElementById('app')! })
