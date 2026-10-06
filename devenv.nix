{ pkgs, ... }:

{
  languages.go = {
    enable = true;
    package = pkgs.go_1_27;
  };

  git-hooks.hooks = {
    gofmt.enable = true;

    go-vet = {
      enable = true;
      name = "go vet";
      entry = "go vet ./...";
      files = "\\.go$";
      pass_filenames = false;
    };

    adr-numbers = {
      enable = true;
      name = "ADR numbers unique";
      entry = "scripts/check-adr-numbers.sh";
      files = "^docs/adr/";
      pass_filenames = false;
    };

    go-test = {
      enable = true;
      name = "go test";
      entry = "go test ./...";
      stages = [ "pre-push" ];
      pass_filenames = false;
    };
  };
}
