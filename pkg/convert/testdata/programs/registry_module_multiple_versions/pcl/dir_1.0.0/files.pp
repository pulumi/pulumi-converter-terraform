allFilePaths    = notImplemented("fileset(var.base_dir,\"**\")")
staticFilePaths = notImplemented("toset([\nforpinlocal.all_file_paths:p\niflength(p)<length(var.template_file_suffix)||substr(p,length(p)-length(var.template_file_suffix),length(var.template_file_suffix))!=var.template_file_suffix\n])")
templateFilePaths = { for p in allFilePaths : invoke("std:index:substr", {
  input  = p
  length = 0
  offset = length(p) - length(templateFileSuffix)
  }).result => p if !invoke("std:index:contains", {
  input   = staticFilePaths
  element = p
}).result }
templateFileContents = { for p, sp in templateFilePaths : p => notImplemented("templatefile(\"$${var.base_dir}/$${sp}\",var.template_vars)") }
staticFileLocalPaths = { for p in staticFilePaths : p => "${baseDir}/${p}" }
outputFilePaths      = notImplemented("setunion(keys(local.template_file_paths),local.static_file_paths)")
fileSuffixMatches    = { for p in outputFilePaths : p => notImplemented("regexall(\"\\\\.[^\\\\.]+\\\\z\",p)") }
fileSuffixes         = { for p, ms in fileSuffixMatches : p => length(ms) > 0 ? ms[0] : "" }
myFileTypes = { for p in outputFilePaths : p => invoke("std:index:lookup", {
  map     = fileTypes
  key     = fileSuffixes[p]
  default = defaultFileType
}).result }
files = invoke("std:index:merge", {
  input = [{ for p in notImplemented("keys(local.template_file_paths)") : p => {
    contentType = myFileTypes[p]
    sourcePath  = notImplemented("tostring(null)")
    content     = templateFileContents[p]
    digests     = notImplemented("tomap({\nmd5=md5(local.template_file_contents[p])\nsha1=sha1(local.template_file_contents[p])\nsha256=sha256(local.template_file_contents[p])\nsha512=sha512(local.template_file_contents[p])\nbase64sha256=base64sha256(local.template_file_contents[p])\nbase64sha512=base64sha512(local.template_file_contents[p])\n})")
    } }, { for p in staticFilePaths : p => {
    contentType = myFileTypes[p]
    sourcePath  = staticFileLocalPaths[p]
    content     = notImplemented("tostring(null)")
    digests     = notImplemented("tomap({\nmd5=filemd5(local.static_file_local_paths[p])\nsha1=filesha1(local.static_file_local_paths[p])\nsha256=filesha256(local.static_file_local_paths[p])\nsha512=filesha512(local.static_file_local_paths[p])\nbase64sha256=filebase64sha256(local.static_file_local_paths[p])\nbase64sha512=filebase64sha512(local.static_file_local_paths[p])\n})")
  } }]
}).result
