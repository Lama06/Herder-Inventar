const Anmelden = {
    emits: ["erfolg"],
    data() {
        return {
            benutzername: "",
            passwort: "",
            passwortSichtbar: false,
        }
    },
    computed: {
        passwortInputType() {
            if (this.passwortSichtbar) {
                return "text"
            }
            return "password"
        },
        absendenKnopfAktiviert() {
            return this.benutzername !== "" && this.passwort !== ""
        }
    },
    methods: {
        async absenden() {
            let response;
            try {
                response = await fetch("/api/auth/login/", {
                    method: "POST",
                    body: JSON.stringify({
                        benutzername: this.benutzername,
                        passwort: this.passwort,
                    })
                })
            } catch {
                alert("Sever antwortet nicht")
                return
            }

            if (response.status === 401) {
                alert("Falsches Anmeldedaten")
                return
            }

            if (!response.ok) {
                alert("Serverfehler")
                return
            }

            let daten;
            try {
                daten = await response.json()
            } catch {
                alert("Server antwortet inkorrekt")
                return
            }

            sessionStorage.setItem("schlüssel", daten.schlüssel)
            this.$emit("erfolg")
        }
    },
    template: `
<h1>Anmelden</h1>
<label for="benutzername-eingabe">Benutzername:</label><input type="text" id="benutzername-eingabe" v-model="benutzername">
<br>
<label for="passwort-eingabe">Passwort: </label>
<input :type="passwortInputType" id="passwort-eingabe" v-model="passwort">
<button @click="passwortSichtbar = !passwortSichtbar">{{ passwortSichtbar ? "Passwort verstecken" : "Passwort anzeigen" }}</button>
<br>
<button :disabled="!absendenKnopfAktiviert" @click="absenden">Abenden</button>
`
}

const InventarEintrag = {
    props: ["id", "name", "raum", "probleme"],
    emits: ["loeschen"],
    data() {
        return {
            nameServer: this.name,
            raumServer: this.raum,

            nameEingabe: this.name,
            raumEingabe: this.raum,
        }
    },
    computed: {
        aenderungenGemacht() {
            return this.nameEingabe !== this.nameServer || this.raumEingabe !== this.raumServer
        }
    },
    methods: {
        async loeschen() {
            let response;
            try {
                response = await fetch("/api/objekte/loeschen/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                        id: this.id
                    })
                })
            } catch {
                alert("Server nicht erreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler")
                return
            }

            this.$emit("loeschen")
        },
        async aenderungenSpeichern() {
            let response;
            try {
                response = await fetch("/api/objekte/aendern/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                        id: this.id,
                        name: this.nameEingabe,
                        raum: this.raumEingabe,
                    })
                })
            } catch {
                alert("Server nicht erreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler");
                return
            }

            this.nameServer = this.nameEingabe
            this.raumServer = this.raumEingabe
        }
    },
    template: `
<div>
    <h3 :title="id">{{ nameServer }}</h3>
    <span :title="id">Name: </span> <input type="text" v-model="nameEingabe">
    <br>

    Raum: <input type="text" v-model="raumEingabe">
    <br>
    
    <template v-if="probleme.length != 0">
        Es gibt {{ probleme.length }} Probleme!
        <br>
    </template>
    
    <button @click="loeschen">Löschen</button>
    <button v-if="aenderungenGemacht" @click="aenderungenSpeichern">Änderungen speichern</button>
    <hr>
</div>
`
}

const Inventar = {
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
                name: "",
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
        v-for="objekt in objekte"
        :key="objekt.id"
        :id="objekt.id"
        :name="objekt.name"
        :raum="objekt.raum"
        :probleme="objekt.probleme"
        @loeschen="loeschen(objekt.id)"
    ></InventarEintrag>
</template>`
}

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