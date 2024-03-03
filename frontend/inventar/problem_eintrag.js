export default {
    props: ["problem", "obj", "index"],
    emits: ["loesen"],
    computed: {
        datumAnzeige() {
            return new Date(this.problem.datum * 1000).toDateString()
        }
    },
    methods: {
        async loesen() {
            let response;
            try {
                response = await fetch("/api/probleme/loesen/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                        id: this.obj.id,
                        problem: this.problem.id,
                    })
                })
            } catch (err) {
                console.log(err)
                alert("Server nicht erreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler")
                return
            }

            this.$emit("loesen")
        }
    },
    template: `
<div>
<h3 :title="problem.id">Problem {{ index + 1 }}</h3>
Erstellt von: {{ problem.ersteller }}
<br>
Datum: {{ datumAnzeige }}
<br>
Beschreibung: {{ problem.beschreibung }}
<button @click="loesen">Lösen</button>
</div>
`
}