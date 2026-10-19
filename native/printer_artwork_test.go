package main

import (
    "bytes"
    "encoding/base64"
    "image"
    "image/color"
    "image/png"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func writeSampleCover(t *testing.T, root, rel string) string {
    t.Helper()
    dest := filepath.Join(root, filepath.FromSlash(rel))
    if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil { t.Fatal(err) }
    picture := image.NewRGBA(image.Rect(0, 0, 128, 96))
    for y:=0; y<96; y++ {
        for x:=0; x<128; x++ { picture.Set(x,y,color.RGBA{25,120,90,255}) }
    }
    var buffer bytes.Buffer
    if err := png.Encode(&buffer, picture); err != nil { t.Fatal(err) }
    if err := os.WriteFile(dest, buffer.Bytes(), 0600); err != nil { t.Fatal(err) }
    return dest
}

func TestLocalCoverUsesInstalledSlicerOnly(t *testing.T) {
    root:=t.TempDir()
    rel:="Snapmaker/Snapmaker U1_cover.png"
    writeSampleCover(t, root, rel)
    uri,ok:=localPNGDataURI(root,rel)
    if !ok || !strings.HasPrefix(uri,"data:image/png;base64,") { t.Fatal("installed PNG rejected") }
    data,err:=base64.StdEncoding.DecodeString(strings.TrimPrefix(uri,"data:image/png;base64,"))
    if err!=nil || !bytes.HasPrefix(data,[]byte("\x89PNG\r\n\x1a\n")) { t.Fatalf("bad PNG data: %v",err) }
    model:=CatalogPrinter{ID:"snapmaker-u1",LocalCoverCandidates:[]string{rel}}
    art:=findPrinterArtwork(model,[]string{root})
    if !art.Found || art.Provenance!="local_installed_slicer" || art.DataURI!=uri {
        t.Fatalf("artwork reply failed: %+v",art)
    }
    absent:=findPrinterArtwork(model,[]string{filepath.Join(root,"not-installed")})
    if absent.Found || absent.DataURI!="" { t.Fatal("must fall back when slicer is missing") }
}

func TestCoverPathSecurityAndNoArbitraryFileRead(t *testing.T) {
    for _,value:=range []string{
        "../../secret_cover.png", "/etc/passwd_cover.png",
        "Vendor/../secret_cover.png", "Vendor\\machine_cover.png",
        "Vendor/machine.jpg", "Vendor/machine.svg",
        "C:/Windows_cover.png", "Vendor/evil\n_cover.png",
        "Vendor/foo/bar_cover.png",
    } {
        if validLocalCoverPath(value) { t.Fatalf("unsafe cover path accepted: %q",value) }
    }
    root:=t.TempDir()
    file:=writeSampleCover(t,root,"Qidi/Qidi X-Plus 5_cover.png")
    if _,ok:=localPNGDataURI(root,"Qidi/Qidi X-Plus 5_cover.png");!ok {
        t.Fatal("valid manufacturer image rejected")
    }
    // Non-image bytes renamed to _cover.png must never reach the browser.
    if err:=os.WriteFile(file,[]byte("<svg><script>alert(1)</script></svg>"),0600);err!=nil {t.Fatal(err)}
    if uri,ok:=localPNGDataURI(root,"Qidi/Qidi X-Plus 5_cover.png");ok||uri!="" {
        t.Fatal("untrusted SVG masquerading as PNG accepted")
    }
    // A symlink to a file outside the installed slicer root must be rejected.
    outside:=t.TempDir()
    external:=writeSampleCover(t,outside,"leak_cover.png")
    link:=filepath.Join(root,"Qidi","linked_cover.png")
    if err:=os.Symlink(external,link);err==nil {
        if uri,ok:=localPNGDataURI(root,"Qidi/linked_cover.png");ok||uri!="" {
            t.Fatal("symlink escaped installed-slicer resource root")
        }
    }
}

func TestArtworkLookupModelIDNotPath(t *testing.T) {
    app:=&App{}
    for _,query:=range []string{"../../settings.json","C:/Users/secret","", "broken\\id"} {
        if _,err:=app.PrinterArtwork(query);err==nil {
            t.Fatalf("untrusted filename used as model ID: %q",query)
        }
    }
    got,err:=app.PrinterArtwork("definitely-not-real-printer")
    if err==nil||got.Found { t.Fatal("unknown printer must never open files") }
}

func TestPrinterCatalogueCoverReferencesAreLocalHintsNotBundledImages(t *testing.T) {
    catalog,err:=catalogSnapshot()
    if err!=nil {t.Fatal(err)}
    expected:=map[string]string{
        "snapmaker-u1":"Snapmaker/Snapmaker U1_cover.png",
        "flashforge-creator-5":"Flashforge/Flashforge Creator 5_cover.png",
        "elegoo-centauri-carbon-2":"Elegoo/Elegoo Centauri Carbon 2_cover.png",
        "qidi-x-plus-5":"X 5 Series/X-Plus 5_cover.png",
    }
    verified:=0
    for _,model:=range catalog.Models {
        for _,ref:=range model.LocalCoverCandidates {
            if !validLocalCoverPath(ref) {t.Fatalf("bad shipped path: %s",ref)}
        }
        if wanted,exists:=expected[model.ID];exists {
            for _,ref:=range model.LocalCoverCandidates {
                if ref==wanted {verified++;break}
            }
        }
        if model.ImagePath!=nil || model.LogoPath!=nil {
            t.Fatal("actual profile pictures may not be shipped")
        }
    }
    if verified!=len(expected){t.Fatalf("only %d/%d known local covers hinted",verified,len(expected))}
}
