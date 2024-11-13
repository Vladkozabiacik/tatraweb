function toggleWorksiteField() {
    var position = document.getElementById("position").value;
    var worksiteDiv = document.getElementById("worksiteDiv");

    if (position === "vyroba") {
        worksiteDiv.classList.remove("hidden");
    } else {
        worksiteDiv.classList.add("hidden");
    }
}