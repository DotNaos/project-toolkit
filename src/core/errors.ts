export class ProjectToolkitError extends Error {
  readonly exitCode: number;

  constructor(message: string, exitCode = 1) {
    super(message);
    this.name = "ProjectToolkitError";
    this.exitCode = exitCode;
  }
}
