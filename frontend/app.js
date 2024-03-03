import Anmelden from "./anmelden.js"
import Inventar from "./inventar/inventar.js"

export default {
    components: {
        Anmelden,
        Inventar
    },
    data() {
        return {
            unterseite: "anmelden"
        }
    },
    methods: {
        angemeldet() {
            this.unterseite = "inventar"
        }
    },
    template: `
<Anmelden v-if="unterseite === 'anmelden'" @erfolg="angemeldet"></Anmelden>
<Inventar v-if="unterseite === 'inventar'"></Inventar>
`
}