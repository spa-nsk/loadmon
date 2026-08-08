<div class="container left15px">
  <div class="row">
    <div class="hero-text">
      <h3>Детали загрузке сервера {{.Filter.ServerName}} с {{printf "%02d.%02d.%4d" .Filter.FromTime.Day .Filter.FromTime.Month .Filter.FromTime.Year}} по {{printf "%02d.%02d.%4d" .Filter.ToTime.Day .Filter.ToTime.Month .Filter.ToTime.Year}}</h3>
    </div>
  </div>
</div>
{{if .DetailsCPU}}
<img class="img-responsive center-block2" alt="Детали загрузке процессора сервера {{.Filter.ServerName}}" title="Детали загрузке сервера {{.Filter.ServerName}}" src="data:image/png;base64,{{.DetailsCPU}}">
<br/>
{{end}}
{{if .DetailsMEM}}
<img class="img-responsive center-block2" alt="Детали загрузке памяти сервера {{.Filter.ServerName}}" title="Детали загрузке сервера {{.Filter.ServerName}}" src="data:image/png;base64,{{.DetailsMEM}}">
<br/>
{{end}}
{{if .DetailsDiskLoad}}
<img class="img-responsive center-block2" alt="Детали загрузке дисков сервера {{.Filter.ServerName}}" title="Детали загрузке сервера {{.Filter.ServerName}}" src="data:image/png;base64,{{.DetailsDiskLoad}}">
<br/>
{{end}}
{{if .DetailsDiskUse}}
<img class="img-responsive center-block2" alt="Детали по заполнению дисков на сервере {{.Filter.ServerName}}" title="Детали загрузке сервера {{.Filter.ServerName}}" src="data:image/png;base64,
{{.DetailsDiskUse}}">
<br/>
{{end}}
{{if .DetailsMIX}}
<img class="img-responsive center-block2" alt="Детали загрузке сервера {{.Filter.ServerName}}" title="Детали загрузке сервера {{.Filter.ServerName}}" src="data:image/png;base64,{{.DetailsMIX}}">
<br/>
{{end}}
