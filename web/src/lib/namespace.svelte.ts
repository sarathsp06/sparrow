import { browser } from '$app/environment';

class NamespaceState {
    current = $state(browser ? localStorage.getItem('sparrow_namespace') || 'default' : 'default');

    setNamespace(value: string) {
        this.current = value;
        if (browser) {
            localStorage.setItem('sparrow_namespace', value);
        }
    }
}

export const namespaceState = new NamespaceState();
