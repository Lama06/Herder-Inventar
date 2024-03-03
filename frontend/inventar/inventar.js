import InventarEintrag from "./eintrag.js"

export default {
    components: {
        InventarEintrag
    },
    data() {
        return {
            objekte: null
        }
    },
    computed: {
        geladen() {
            return this.objekte !== null
        }
    },
    methods: {
        loeschen(geloescht) {
            this.objekte = this.objekte.filter(obj => obj.id !== geloescht)
        },
        aendern(index, neu) {
            this.objekte[index] = neu
        },
        async add() {
            let response
            try {
                response = await fetch("/api/objekte/erstellen/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                    })
                })
            } catch {
                alert("Server unerreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler")
                return
            }

            let daten
            try {
                daten = await response.json()
            } catch {
                alert("Fehler")
                return
            }

            this.objekte.unshift({
                id: daten.id,
                name: "Neu erstellt",
                raum: "",
                probleme: [],
            })
        }
    },
    async mounted() {
        let response
        try {
            response = await fetch("/api/objekte/auflisten/", {
                method: "POST",
                body: JSON.stringify({
                    schlüssel: sessionStorage.getItem("schlüssel"),
                })
            })
        } catch {
            alert("Server unerreichbar");
            return
        }

        if (!response.ok) {
            alert("Fehler")
            return
        }

        let text
        try {
            text = await response.json()
        } catch {
            alert("Fehler")
            return
        }

        for (let objekt of text.objekte) {
            if (objekt.probleme == null) {
                objekt.probleme = []
            }
        }

        this.objekte = text.objekte
    },
    template: `
<template v-if="!geladen">
<h1>Laden...</h1>
</template>
<template v-else>
    <h1>Inventar des Herders ({{ objekte.length }} Objekte)</h1>
    <button @click="add">Objekt hinzufügen</button>
    <br>
    <InventarEintrag 
        v-for="(objekt, index) in objekte"
        :key="objekt.id"
        :obj="objekt"
        @loeschen="loeschen(objekt.id)"
        @aendern="(obj) => aendern(index, obj)"
    ></InventarEintrag>
</template>`
}